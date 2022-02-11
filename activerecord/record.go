package activerecord

import (
	"context"
	"fmt"
	"strings"

	. "github.com/activegraph/activegraph/activesupport"
)

type ErrRecordNotFound struct {
	PrimaryKey string
	ID         interface{}
}

func (e *ErrRecordNotFound) Is(target error) bool {
	_, ok := target.(*ErrRecordNotFound)
	return ok
}

func (e *ErrRecordNotFound) Error() string {
	return fmt.Sprintf("record not found by %s = %v", e.PrimaryKey, e.ID)
}

type ErrRecordNotUnique struct {
	Err error
}

func (e *ErrRecordNotUnique) Is(target error) bool {
	_, ok := target.(*ErrRecordNotUnique)
	return ok
}

func (e *ErrRecordNotUnique) Error() string {
	return e.Err.Error()
}

type CollectionResult struct {
	Result[*Relation]
}

func OkCollection(rel *Relation) CollectionResult {
	return CollectionResult{Ok(rel)}
}

func ErrCollection(err error) CollectionResult {
	return CollectionResult{Err[*Relation](err)}
}

func (c CollectionResult) ToA() (Array, error) {
	if c.IsErr() {
		return nil, c.Err()
	}
	return c.Unwrap().ToA()
}

func (c CollectionResult) DeleteAll() error {
	records, err := c.ToA()
	if err != nil {
		return err
	}
	for i := 0; i < len(records); i++ {
		if _, err := records[i].Delete(); err != nil {
			return err
		}
	}
	return nil
}

type RecordResult struct {
	Result[*ActiveRecord]
}

func OkRecord(r *ActiveRecord) RecordResult {
	return RecordResult{Ok(r)}
}

func ErrRecord(err error) RecordResult {
	return RecordResult{Err[*ActiveRecord](err)}
}

func ReturnRecord(r *ActiveRecord, err error) RecordResult {
	return RecordResult{Return(r, err)}
}

func (r RecordResult) andThen(op func(*ActiveRecord) (*ActiveRecord, error)) RecordResult {
	return RecordResult{r.AndThen(func(r *ActiveRecord) Result[*ActiveRecord] {
		var (
			rec *ActiveRecord
			err error
		)
		if r != nil {
			rec, err = op(r)
		}
		return Return(rec, err)
	})}
}

func (r RecordResult) Insert() RecordResult {
	return r.andThen((*ActiveRecord).Insert)
}

func (r RecordResult) Update() RecordResult {
	return r.andThen((*ActiveRecord).Update)
}

func (r RecordResult) Delete() RecordResult {
	return r.andThen((*ActiveRecord).Delete)
}

func (r RecordResult) Association(name string) RecordResult {
	return RecordResult{r.AndThen(func(r *ActiveRecord) Result[*ActiveRecord] {
		return r.Association(name)
	})}
}

func (r RecordResult) AssignAssociation(name string, target RecordResult) RecordResult {
	return RecordResult{r.AndThen(func(r *ActiveRecord) Result[*ActiveRecord] {
		if target.IsErr() {
			return target
		}
		err := r.associations.AssignAssociation(name, target.Unwrap())
		return ReturnRecord(r, err)
	})}
}

func (r RecordResult) AssignCollection(name string, targets ...RecordResult) RecordResult {
	return RecordResult{r.AndThen(func(r *ActiveRecord) Result[*ActiveRecord] {
		records := make([]*ActiveRecord, 0, len(targets))
		for i := 0; i < len(targets); i++ {
			if targets[i].IsErr() {
				return targets[i]
			}
			records = append(records, targets[i].Unwrap())
		}
		err := r.associations.AssignCollection(name, records...)
		return ReturnRecord(r, err)
	})}
}

func (r RecordResult) Collection(name string) CollectionResult {
	if rec := r.Ok(); rec.IsSome() {
		return rec.Unwrap().Collection(name)
	}
	if r.IsErr() {
		return ErrCollection(r.Err())
	}
	return OkCollection(nil)
}

type ActiveRecord struct {
	name      string
	tableName string
	conn      Conn
	ctx       context.Context

	attributes *attributes
	AttributeMethods
	AttributeAccessors

	validations

	associations *associations
	AssociationMethods
	AssociationAccessors
	CollectionAccessors
}

func (r *ActiveRecord) init() *ActiveRecord {
	r.AttributeMethods = r.attributes
	r.AttributeAccessors = r.attributes

	r.associations.delegateAccessors(r)

	r.AssociationMethods = r.associations
	r.AssociationAccessors = r.associations
	r.CollectionAccessors = r.associations
	return r
}

func (r *ActiveRecord) ToHash() Hash {
	hash := make(Hash, len(r.attributes.keys))
	for key := range r.attributes.keys {
		hash[key] = nil
		if value, ok := r.attributes.values[key]; ok {
			hash[key] = value
		}
	}
	return hash
}

func (r *ActiveRecord) Name() string {
	return r.name
}

func (r *ActiveRecord) Copy() *ActiveRecord {
	return (&ActiveRecord{
		name:         r.name,
		tableName:    r.tableName,
		conn:         r.conn,
		ctx:          r.ctx,
		attributes:   r.attributes.copy(),
		associations: r.associations.copy(),
	}).init()
}

func (r *ActiveRecord) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

func (r *ActiveRecord) WithContext(ctx context.Context) *ActiveRecord {
	newr := r.Copy()
	newr.ctx = ctx
	return newr
}

func (r *ActiveRecord) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "#<%s ", strings.Title(r.name))

	attrNames := r.AttributeNames()
	for i, attrName := range attrNames {
		fmt.Fprintf(&buf, "%s: %#v", attrName, r.Attribute(attrName))
		if i < len(attrNames)-1 {
			fmt.Fprint(&buf, ", ")
		}
	}

	fmt.Fprintf(&buf, ">")
	return buf.String()
}

// IsValid runs all the validations, returns true if no errors are found, false othewrise.
// Alias for Validate.
func (r *ActiveRecord) IsValid() bool {
	return r.Validate() == nil
}

// Validate runs all the validation, returns unpassed validations, nil otherwise.
func (r *ActiveRecord) Validate() error {
	return r.validations.validate(r)
}

func (r *ActiveRecord) Insert() (*ActiveRecord, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	columnValues := make([]ColumnValue, 0, len(r.attributes.values))
	for name, value := range r.attributes.values {
		columnValue := ColumnValue{
			Name:  name,
			Type:  r.attributes.keys[name].AttributeType(),
			Value: value,
		}
		columnValues = append(columnValues, columnValue)
	}
	op := InsertOperation{
		TableName:    r.tableName,
		ColumnValues: columnValues,
	}

	id, err := r.conn.ExecInsert(r.Context(), &op)
	if err != nil {
		return nil, err
	}

	err = r.AssignAttribute(r.attributes.primaryKey.AttributeName(), id)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *ActiveRecord) Update() (*ActiveRecord, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	columnValues := make([]ColumnValue, 0, len(r.attributes.values))
	for name, value := range r.attributes.values {
		columnValue := ColumnValue{
			Name:  name,
			Type:  r.attributes.keys[name].AttributeType(),
			Value: value,
		}
		columnValues = append(columnValues, columnValue)
	}

	op := UpdateOperation{
		TableName:    r.tableName,
		PrimaryKey:   r.attributes.primaryKey.AttributeName(),
		ColumnValues: columnValues,
	}

	return r, r.conn.ExecUpdate(r.Context(), &op)
}

func (r *ActiveRecord) Delete() (*ActiveRecord, error) {
	op := DeleteOperation{
		TableName:  r.tableName,
		PrimaryKey: r.attributes.primaryKey.AttributeName(),
		Value:      r.ID(),
	}

	err := r.conn.ExecDelete(r.Context(), &op)
	if err != nil {
		return nil, err
	}
	return r, nil
}
