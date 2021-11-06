package activerecord

import (
	"context"
	"fmt"
	"strings"

	"github.com/activegraph/activegraph/activesupport"
)

type ErrRecordNotFound struct {
	PrimaryKey string
	ID         interface{}
}

func (e ErrRecordNotFound) Error() string {
	return fmt.Sprintf("record not found by %s = %v", e.PrimaryKey, e.ID)
}

type CollectionResult interface {
	activesupport.Result

	UnwrapRelation() *Relation

	ToA() Array

	// QueryMethods
}

func ReturnCollection(rel *Relation, err error) CollectionResult {
	return collectionResult{activesupport.Return(rel, err)}
}

func OkCollection(rel *Relation) CollectionResult {
	return collectionResult{activesupport.Ok(rel)}
}

func ErrCollection(err error) CollectionResult {
	return collectionResult{activesupport.Err(err)}
}

type collectionResult struct {
	activesupport.SomeResult
}

func (r collectionResult) andThen(op func(*Relation) (*Relation, error)) CollectionResult {
	if r.IsOk() {
		return ReturnCollection(op(r.Ok().(*Relation)))
	}
	return r
}

func (r collectionResult) UnwrapRelation() *Relation {
	return r.Unwrap().(*Relation)
}

func (r collectionResult) ToA() Array {
	arr, _ := r.UnwrapRelation().ToA()
	return arr
}

func (r collectionResult) Len() int {
	return 0
}

type Result interface {
	activesupport.Result

	UnwrapRecord() *ActiveRecord

	Insert() Result
	Update() Result
	Delete() Result

	// TODO:
	// Upsert() Record
	// Destroy() Record

	// AttributeMethods
	// AttributeAccessors

	// AssociationMethods
	// AggregationMethods
}

func Return(r *ActiveRecord, err error) Result {
	return result{activesupport.Return(r, err)}
}

func Ok(r *ActiveRecord) Result {
	return result{activesupport.Ok(r)}
}

func Err(err error) Result {
	return result{activesupport.Err(err)}
}

type result struct {
	activesupport.SomeResult
}

func (r result) UnwrapRecord() *ActiveRecord {
	return r.Unwrap().(*ActiveRecord)
}

func (r result) andThen(op func(*ActiveRecord) (*ActiveRecord, error)) Result {
	if r.IsOk() {
		return Return(op(r.Ok().(*ActiveRecord)))
	}
	return r
}

func (r result) Insert() Result {
	return r.andThen((*ActiveRecord).Insert)
}

func (r result) Update() Result {
	return r.andThen((*ActiveRecord).Update)
}

func (r result) Delete() Result {
	return r.andThen((*ActiveRecord).Delete)
}

type ActiveRecord struct {
	name      string
	tableName string
	conn      Conn
	ctx       context.Context

	attributes
	validations

	associations *associations
	AssociationMethods
	AssociationAccessors
	CollectionAccessors
}

func (r *ActiveRecord) init() *ActiveRecord {
	r.associations.delegateAccessors(r)

	r.AssociationMethods = r.associations
	r.AssociationAccessors = r.associations
	r.CollectionAccessors = r.associations
	return r
}

func (r *ActiveRecord) ToHash() activesupport.Hash {
	hash := make(activesupport.Hash, len(r.attributes.keys))
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
		attributes:   *r.attributes.copy(),
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
		fmt.Fprintf(&buf, "%s: %#v", attrName, r.AccessAttribute(attrName))
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

func (r *ActiveRecord) AssignAssociation(assocName string, rec *ActiveRecord) error {
	if !r.HasAssociation(assocName) {
		return ErrUnknownAssociation{RecordName: r.name, Assoc: assocName}
	}

	return nil
}

// Association returns the associated object, nil is returned if none is found.
func (r *ActiveRecord) Association(assocName string) *ActiveRecord {
	return r.AccessAssociation(assocName).UnwrapRecord()
}

// Collection returns a Relation of all associated records. A nil is returned
// if relation does not belong to the record.
func (r *ActiveRecord) Collection(assocName string) CollectionResult {
	return r.AccessCollection(assocName)
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

	err = r.AssignAttribute(r.primaryKey.AttributeName(), id)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *ActiveRecord) Update() (*ActiveRecord, error) {
	return nil, nil
}

func (r *ActiveRecord) Delete() (*ActiveRecord, error) {
	op := DeleteOperation{
		TableName:  r.tableName,
		PrimaryKey: r.primaryKey.AttributeName(),
		Value:      r.ID(),
	}

	err := r.conn.ExecDelete(r.Context(), &op)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *ActiveRecord) IsPersisted() bool {
	return false
}
