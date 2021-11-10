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

type CollectionOption struct {
	activesupport.Option
}

// NoneCollection is no Collection value.
var NoneCollection = CollectionOption{activesupport.None(_CollectionType)}

// SomeCollection is a value of Relation.
func SomeCollection(r *Relation) CollectionOption {
	return CollectionOption{activesupport.Some(_CollectionType, r)}
}

func (o CollectionOption) Unwrap() *Relation {
	return o.Option.Unwrap().(*Relation)
}

func (o CollectionOption) UnwrapOrDefault() *Relation {
	return o.Option.UnwrapOrDefault().(*Relation)
}

type CollectionResult struct {
	result activesupport.Result
}

func ReturnCollection(rel *Relation, err error) CollectionResult {
	switch rel {
	case nil:
		return CollectionResult{
			activesupport.Return(_CollectionType, NoneCollection, err),
		}
	default:
		return CollectionResult{
			activesupport.Return(_CollectionType, SomeCollection(rel), err),
		}
	}
}

func OkCollection(option CollectionOption) CollectionResult {
	return CollectionResult{activesupport.Ok(_CollectionType, option)}
}

func ErrCollection(err error) CollectionResult {
	return ReturnCollection(nil, err)
}

func (r CollectionResult) Expect(msg string) *Relation {
	return r.result.Expect(msg).(CollectionOption).Unwrap()
}
func (r CollectionResult) ExpectErr(msg string) error   { return r.result.ExpectErr(msg) }
func (r CollectionResult) Result() activesupport.Result { return r.result }
func (r CollectionResult) String() string               { return r.result.String() }
func (r CollectionResult) IsErr() bool                  { return r.result.IsErr() }
func (r CollectionResult) Err() error                   { return r.result.Err() }

func (r CollectionResult) Ok() CollectionOption {
	return r.result.Ok().UnwrapOr(NoneCollection).(CollectionOption)
}

func (r CollectionResult) Unwrap() *Relation {
	return r.Ok().Unwrap()
}

// func (r CollectionResult) AndThen(op func(CollectionOption) CollectionResult) CollectionResult {
// 	result := r.Result.AndThen(func(val interface{}) activesupport.Result {
// 		return op(val.(CollectionOption)).Result
// 	})
// 	return CollectionResult{result}
// }

func (r CollectionResult) ToA() (Array, error) {
	if r.IsErr() {
		return nil, r.Err()
	}
	return r.Unwrap().ToA()
}

func (r CollectionResult) Len() int {
	return 0
}

var (
	// _ AssociationAccessors = Result{}
	// _ CollectionAccessors  = Result{}

	_Type           = new(_TypeImpl)
	_CollectionType = new(_CollectionTypeImpl)
)

type _TypeImpl struct{}

func (*_TypeImpl) Default() interface{} { return (*ActiveRecord)(nil) }

type _CollectionTypeImpl struct{}

func (*_CollectionTypeImpl) Default() interface{} { return (*Relation)(nil) }

type Option struct {
	activesupport.Option
}

// None is no ActiveRecord value.
var None = Option{activesupport.None(_Type)}

// Some value of ActiveRecord.
func Some(r *ActiveRecord) Option {
	return Option{activesupport.Some(_Type, r)}
}

func (o Option) Unwrap() *ActiveRecord {
	return o.Option.Unwrap().(*ActiveRecord)
}

func (o Option) UnwrapOrDefault() *ActiveRecord {
	return o.Option.UnwrapOrDefault().(*ActiveRecord)
}

func Return(r *ActiveRecord, err error) Result {
	switch r {
	case nil:
		return Result{activesupport.Return(_Type, None, err)}
	default:
		return Result{activesupport.Return(_Type, Some(r), err)}
	}
}

func Ok(option Option) Result {
	return Result{activesupport.Ok(_Type, option)}
}

func Err(err error) Result {
	return Return(nil, err)
}

type Result struct {
	result activesupport.Result
}

func (r Result) Result() activesupport.Result { return r.result }
func (r Result) String() string               { return r.result.String() }
func (r Result) IsErr() bool                  { return r.result.IsErr() }
func (r Result) Err() error                   { return r.result.Err() }

func (r Result) UnwrapOr(rec *ActiveRecord) *ActiveRecord {
	return r.result.UnwrapOr(Some(rec)).(Option).Unwrap()
}

func (r Result) Expect(msg string) *ActiveRecord { return r.result.Expect(msg).(Option).Unwrap() }
func (r Result) ExpectErr(msg string) error      { return r.result.ExpectErr(msg) }

func (r Result) Ok() Option {
	return r.result.Ok().UnwrapOr(None).(Option)
}

func (r Result) Unwrap() *ActiveRecord {
	return r.Ok().Unwrap()
}

// func (r *Result) AttributePresent(attrName string) bool {
// }
// func (r *Result) Attribute(attrName string) interface{} {
// }
// func (r *Result) AccessAttribute(attrName string) interface{} {
// }
// func (r *Result) AssignAttribute(attrName string, val interface{}) error {
// }
// func (r *Result) AssignAttributes(newAttributes map[string]interface{}) error {
// }

func (r Result) AndThen(op func(Option) Result) Result {
	result := r.result.AndThen(func(val interface{}) activesupport.Result {
		return op(val.(Option)).result
	})
	return Result{result}
}

func (r Result) Insert() Result {
	return r.AndThen(func(o Option) Result {
		if o.IsNone() {
			return Ok(None)
		}
		return Return(o.Unwrap().Insert())
	})
}
func (r Result) Update() Result { return Err(fmt.Errorf("not implemented")) }
func (r Result) Delete() Result { return Err(fmt.Errorf("not implemented")) }

func (r Result) Association(name string) Result {
	switch r.Ok() {
	case None:
		return r
	default:
		return r.Unwrap().Association(name)
	}
}

func (r Result) Collection(name string) CollectionResult {
	switch r.Ok() {
	case None:
		return OkCollection(NoneCollection)
	default:
		return r.Unwrap().Collection(name)
	}
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

func (r *ActiveRecord) AssignAssociation(assocName string, rec *ActiveRecord) error {
	if !r.HasAssociation(assocName) {
		return ErrUnknownAssociation{RecordName: r.name, Assoc: assocName}
	}

	return nil
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
	return nil, nil
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
