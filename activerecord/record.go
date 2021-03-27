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
	return &ActiveRecord{
		name:       r.name,
		conn:       r.conn,
		ctx:        r.ctx,
		reflection: r.reflection,
		attributes: *r.attributes.copy(),
	}
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

func (r *ActiveRecord) AccessAssociation(assocName string) (*ActiveRecord, error) {
	rel, err := r.reflection.Reflection(assocName)
	if err != nil {
		return nil, err
	}

	assocId := r.AccessAttribute(assocName + "_id")
	return rel.WithContext(r.Context()).Find(assocId)
}

// Association returns the associated object, nil is returned if none is found.
func (r *ActiveRecord) Association(assocName string) *ActiveRecord {
	rec, _ := r.AccessAssociation(assocName)
	return rec
}

func (r *ActiveRecord) AccessCollection(assocName string) (*Relation, error) {
	rel, err := r.reflection.Reflection(assocName)
	if err != nil {
		return nil, err
	}

	rel = rel.WithContext(r.Context())
	err = rel.scope.AssignAttribute(r.name+"_id", r.ID())
	if err != nil {
		return nil, err
	}
	return rel, nil
}

// Collection returns a Relation of all associated records. A `nil` is returned
// if relation does not belong to the record.
func (r *ActiveRecord) Collection(assocName string) *Relation {
	rel, _ := r.AccessCollection(assocName)
	return rel
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
	attrs      attributesMap
	assocs     associationsMap
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
	r.attrs[name+"_id"] = IntAttr{Name: name + "_id", Validates: IntValidators(nil)}
	r.assocs[name] = &BelongsToAssoc{Name: name}
}

func (r *R) HasMany(name string) {
	r.assocs[name] = &HasManyAssoc{Name: name}
}

type Relation struct {
	name       string
	conn       Conn
	scope      *attributes
	query      *query
	reflection *Reflection
	ctx        context.Context
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
		assocs:     make(associationsMap),
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

	// The scope is empty by default.
	scope, err := newAttributes(name, r.attrs.copy(), nil)
	if err != nil {
		return nil, err
	}

	// Create the model schema, and register it within a reflection instance.
	rel := &Relation{
		name:       name,
		scope:      scope,
		reflection: r.reflection,
		query:      new(query),
	}
	r.reflection.AddReflection(name, rel)

	return rel, nil
}

func (rel *Relation) Name() string {
	return rel.name
}

func (rel *Relation) Copy() *Relation {
	return &Relation{
		name:       rel.name,
		conn:       rel.conn,
		scope:      rel.scope.copy(),
		query:      rel.query.copy(),
		reflection: rel.reflection,
		ctx:        rel.ctx,
	}
}

// IsEmpty returns true if there are no records.
func (rel *Relation) IsEmpty() bool {
	// TODO: implement the method.
	return false
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

func (rel *Relation) Connect(conn Conn) {
	rel.conn = conn
}

func (rel *Relation) New(params map[string]interface{}) *ActiveRecord {
	rec, err := rel.Create(params)
	if err != nil {
		panic(err)
	}
	return rec
}

func (rel *Relation) Create(params map[string]interface{}) (*ActiveRecord, error) {
	attributes := rel.scope.clear()
	err := attributes.AssignAttributes(params)
	if err != nil {
		return nil, err
	}

	return &ActiveRecord{
		name:       rel.name,
		conn:       rel.conn,
		attributes: *attributes,
		reflection: rel.reflection,
	}, nil
}

// PrimaryKey returns the attribute name of the record's primary key.
func (rel *Relation) PrimaryKey() string {
	return rel.scope.PrimaryKey()
}

func (rel *Relation) All() *Relation {
	return rel.Copy()
}

func (rel *Relation) Each(fn func(*ActiveRecord) error) error {
	return nil
}

func (rel *Relation) Where(cond string, arg interface{}) *Relation {
	newrel := rel.Copy()

	// When the condition is a regular column, pass it through the regular
	// column comparison instead of query chain predicates.
	if newrel.scope.HasAttribute(cond) {
		newrel.scope.AssignAttribute(cond, arg)
	} else {
		newrel.query.where(cond, arg)
	}
	return newrel
}

func (rel *Relation) Select(attrNames ...string) *Relation {
	newrel := rel.Copy()

	if !newrel.scope.HasAttributes(attrNames...) {
		newrel.scope, _ = newAttributes(rel.name, nil, nil)
		return newrel
	}

	attrMap := make(map[string]struct{}, len(attrNames))
	for _, attrName := range attrNames {
		attrMap[attrName] = struct{}{}
	}

	for _, attrName := range newrel.scope.AttributeNames() {
		if _, ok := attrMap[attrName]; !ok {
			newrel.scope.ExceptAttribute(attrName)
		}
	}
	return newrel
}

func (rel *Relation) Group(attrNames ...string) *Relation {
	newrel := rel.Copy()

	// When the attribute is not part of the scope, return an empty relation.
	if !newrel.scope.HasAttributes(attrNames...) {
		newrel.scope, _ = newAttributes(rel.name, nil, nil)
		return newrel
	}

	newrel.query.group(attrNames...)
	return newrel
}

// ToA converts Relation to array. The method access database to retrieve objects.
func (rel *Relation) ToA() ([]*ActiveRecord, error) {
	op := QueryOperation{
		TableName:   rel.name + "s",
		Columns:     rel.scope.AttributeNames(),
		Values:      make(map[string]interface{}),
		Predicates:  rel.query.predicates,
		GroupValues: rel.query.groupValues,
	}

	// When the scope is configured for the relation, add all attributes
	// to the list of query operation, so only neccessary subset of records
	// are returned to the caller.
	rel.scope.each(func(name string, value interface{}) {
		op.Values[name] = value
	})

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
	op := QueryOperation{
		TableName: rel.name + "s",
		Columns:   rel.scope.AttributeNames(),
		Values:    map[string]interface{}{rel.PrimaryKey(): id},
	}

	rows, err := rel.conn.ExecQuery(rel.Context(), &op)
	if err != nil {
		return nil, err
	}

	if len(rows) != 1 {
		return nil, &ErrRecordNotFound{PrimaryKey: rel.PrimaryKey(), ID: id}
	}
	return rel.Create(rows[0])
}
