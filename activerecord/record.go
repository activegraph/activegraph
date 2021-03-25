package activerecord

import (
	"context"
	"fmt"
	"strings"
)

type ErrRecordNotFound struct {
	PrimaryKey string
	ID         interface{}
}

func (e *ErrRecordNotFound) Error() string {
	return fmt.Sprintf("record not found by %s = %v", e.PrimaryKey, e.ID)
}

type ActiveRecord struct {
	name       string
	conn       Conn
	ctx        context.Context
	reflection *Reflection

	attributes
}

func (r *ActiveRecord) Copy() *ActiveRecord {
	// TODO: implement a shallow copy of the active record.
	return r
}

func (r *ActiveRecord) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

func (r *ActiveRecord) WithContext(ctx context.Context) *ActiveRecord {
	rCopy := r.Copy()
	rCopy.ctx = ctx
	return rCopy
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
	rel, err := r.reflection.Reflection(assocName)
	if err != nil {
		return nil, err
	}

	assocId := r.AccessAttribute(assocName + "_id")
	return rel.WithContext(r.Context()).Find(assocId)
}

func (r *ActiveRecord) Collection(assocName string) (*Relation, error) {
	return r.reflection.Reflection(assocName)
}

func (r *ActiveRecord) Insert() (*ActiveRecord, error) {
	op := InsertOperation{
		// TODO: specify plural name of a record table.
		TableName: r.recordName + "s",
		Values:    r.values,
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

func (r *ActiveRecord) Delete() error {
	op := DeleteOperation{
		TableName:  r.recordName + "s",
		PrimaryKey: r.primaryKey.AttributeName(),
		Value:      r.ID(),
	}

	return r.conn.ExecDelete(r.Context(), &op)
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

type Relation struct {
	name       string
	conn       Conn
	attrs      attributesMap
	reflection *Reflection

	ctx context.Context
}

func New(name string, defineRecord func(*R)) *Relation {
	schema, err := Create(name, defineRecord)
	if err != nil {
		panic(err)
	}
	return schema
}

func Create(name string, defineRecord func(*R)) (*Relation, error) {
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
	model := &Relation{name: name, attrs: r.attrs, reflection: r.reflection}
	r.reflection.AddReflection(name, model)

	return model, nil
}

func (rel *Relation) Copy() *Relation {
	// TODO: implement at least shallow copy of the relation.
	return rel
}

func (rel *Relation) Context() context.Context {
	if rel.ctx == nil {
		return context.Background()
	}
	return rel.ctx
}

func (rel *Relation) WithContext(ctx context.Context) *Relation {
	relCopy := rel.Copy()
	relCopy.ctx = ctx
	return relCopy
}

func (rel *Relation) Connect(conn Conn) *Relation {
	rel.conn = conn
	return rel
}

func (rel *Relation) New(params map[string]interface{}) *ActiveRecord {
	rec, err := rel.Create(params)
	if err != nil {
		panic(err)
	}
	return rec
}

// PrimaryKey returns the attribute name of the record's primary key.
func (rel *Relation) PrimaryKey() string {
	attrs, _ := newAttributes(rel.name, rel.attrs.Copy(), nil)
	return attrs.primaryKey.AttributeName()
}

func (rel *Relation) Create(params map[string]interface{}) (*ActiveRecord, error) {
	attributes, err := newAttributes(rel.name, rel.attrs.Copy(), params)
	if err != nil {
		return nil, err
	}
	return &ActiveRecord{
		name:       rel.name,
		conn:       rel.conn,
		attributes: attributes,
		reflection: rel.reflection,
	}, nil
}

func (rel *Relation) All() ([]*ActiveRecord, error) {
	attrs, err := newAttributes(rel.name, rel.attrs.Copy(), nil)
	if err != nil {
		return nil, err
	}

	op := QueryOperation{
		TableName: rel.name + "s",
		Columns:   attrs.AttributeNames(),
	}

	rows, err := rel.conn.ExecQuery(rel.Context(), &op)
	if err != nil {
		return nil, err
	}

	rr := make([]*ActiveRecord, 0, len(rows))
	for i := range rows {
		rec, err := rel.Create(rows[i])
		if err != nil {
			return nil, err
		}
		rr = append(rr, rec)
	}
	return rr, nil
}

func (rel *Relation) Find(id interface{}) (*ActiveRecord, error) {
	attrs, err := newAttributes(rel.name, rel.attrs.Copy(), nil)
	if err != nil {
		return nil, err
	}

	op := QueryOperation{
		TableName: rel.name + "s",
		Columns:   attrs.AttributeNames(),
		Values:    map[string]interface{}{attrs.PrimaryKey(): id},
	}
	rows, err := rel.conn.ExecQuery(rel.Context(), &op)
	if err != nil {
		return nil, err
	}

	if len(rows) != 1 {
		return nil, &ErrRecordNotFound{PrimaryKey: attrs.PrimaryKey(), ID: id}
	}
	return rel.Create(rows[0])
}
