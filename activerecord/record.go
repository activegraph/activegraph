package activerecord

import (
	"context"
	"fmt"
	"strings"
)

type ActiveRecord struct {
	name       string
	conn       Conn
	reflection *Reflection

	attributes
}

func (r *ActiveRecord) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "#<%s:%p ", strings.Title(r.name), r)

	attrNames := r.AttributeNames()
	for i, attrName := range attrNames {
		fmt.Fprintf(&buf, "%s:%#v", attrName, r.AccessAttribute(attrName))
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
	for attrName, attr := range r.attributes.keys {
		if err := attr.Validate(r.attributes.values[attrName]); err != nil {
			return err
		}
	}
	return nil
}

func (r *ActiveRecord) Association(assocName string) (*ActiveRecord, error) {
	model, err := r.reflection.Reflection(assocName)
	if err != nil {
		return nil, err
	}

	assocId := r.AccessAttribute(assocName + "_id")
	return model.Find(context.TODO(), assocId)
}

func (r *ActiveRecord) Collection(assocName string) ([]*ActiveRecord, error) {
	return nil, nil
}

func (r *ActiveRecord) Insert(ctx context.Context) (*ActiveRecord, error) {
	op := InsertOperation{
		// TODO: specify plural name of a record table.
		TableName: r.recordName + "s",
		Values:    r.values,
	}

	id, err := r.conn.ExecInsert(ctx, &op)
	if err != nil {
		return nil, err
	}

	err = r.AssignAttribute(r.primaryKey.AttributeName(), id)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *ActiveRecord) Update(ctx context.Context) (*ActiveRecord, error) {
	return nil, nil
}

func (r *ActiveRecord) Delete(ctx context.Context) error {
	op := DeleteOperation{
		TableName:  r.recordName + "s",
		PrimaryKey: r.primaryKey.AttributeName(),
		Value:      r.ID(),
	}

	return r.conn.ExecDelete(ctx, &op)
}

func (r *ActiveRecord) IsPersisted() bool {
	return false
}

type ErrUnknownPrimaryKey struct {
	PrimaryKey  string
	Description string
}

func (e *ErrUnknownPrimaryKey) Error() string {
	return fmt.Sprintf("Primary key is unknown, %s", e.Description)
}

type R struct {
	primaryKey string
	attrs      map[string]Attribute
	assocs     map[string]Association
	reflection *Reflection
}

func (r *R) PrimaryKey(name string) {
	r.primaryKey = name
}

func (r *R) AttrInt(name string, validates ...IntValidator) {
	r.attrs[name] = IntAttr{Name: name, Validates: validates}
}

func (r *R) AttrString(name string, validates ...StringValidator) {
	r.attrs[name] = StringAttr{Name: name, Validates: validates}
}

func (r *R) Validates(name string, validators ...Validator) {
}

func (r *R) Scope(reflection *Reflection) {
	if reflection == nil {
		panic("nil reflection")
	}
	r.reflection = reflection
}

func (r *R) BelongsTo(name string) {
	r.attrs[name+"_id"] = StringAttr{Name: name + "_id", Validates: StringValidators(nil)}
	r.assocs[name] = &BelongsToAssoc{Name: name}
}

func (r *R) HasMany(name string) {
}

type ModelSchema struct {
	name       string
	conn       Conn
	attrs      attributesMap
	reflection *Reflection
}

func New(name string, defineRecord func(*R)) *ModelSchema {
	schema, err := Create(name, defineRecord)
	if err != nil {
		panic(err)
	}
	return schema
}

func Create(name string, defineRecord func(*R)) (*ModelSchema, error) {
	r := R{
		assocs:     make(map[string]Association),
		attrs:      make(attributesMap),
		reflection: globalReflection,
	}
	defineRecord(&r)

	// When the primary key was assigned to record builder, mark it explicitely
	// wrapping with PrimaryKey structure. Otherwise, fallback to the default primary
	// key implementation.
	if r.primaryKey != "" {
		attr, ok := r.attrs[r.primaryKey]
		if !ok {
			return nil, &ErrUnknownPrimaryKey{r.primaryKey, "not in attributes"}
		}
		r.attrs[r.primaryKey] = PrimaryKey{Attribute: attr}
	}

	// Create the model schema, and register it within a reflection instance.
	model := &ModelSchema{name: name, attrs: r.attrs, reflection: r.reflection}
	r.reflection.AddReflection(name, model)

	return model, nil
}

// PrimaryKey returns the attribute name of the record's primary key.
func (ms *ModelSchema) PrimaryKey() string {
	attrs, _ := newAttributes(ms.name, ms.attrs.Copy(), nil)
	return attrs.primaryKey.AttributeName()
}

func (ms *ModelSchema) Connect(conn Conn) *ModelSchema {
	ms.conn = conn
	return ms
}

func (ms *ModelSchema) New(params map[string]interface{}) *ActiveRecord {
	rec, err := ms.Create(params)
	if err != nil {
		panic(err)
	}
	return rec
}

func (ms *ModelSchema) Create(params map[string]interface{}) (*ActiveRecord, error) {
	attributes, err := newAttributes(ms.name, ms.attrs.Copy(), params)
	if err != nil {
		return nil, err
	}
	return &ActiveRecord{
		name:       ms.name,
		conn:       ms.conn,
		attributes: attributes,
		reflection: ms.reflection,
	}, nil
}

func (ms *ModelSchema) Find(ctx context.Context, id interface{}) (*ActiveRecord, error) {
	attrs, err := newAttributes(ms.name, ms.attrs.Copy(), nil)
	if err != nil {
		return nil, err
	}

	op := QueryOperation{
		TableName:  ms.name + "s",
		PrimaryKey: attrs.PrimaryKey(),
		Value:      id,
		Columns:    attrs.AttributeNames(),
	}
	cols, err := ms.conn.ExecQuery(ctx, &op)
	if err != nil {
		return nil, err
	}

	return ms.Create(cols)
}
