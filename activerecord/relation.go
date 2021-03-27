package activerecord

import (
	"context"
	"fmt"
)

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

	var lasterr error

	err := rel.conn.ExecQuery(rel.Context(), &op, func(h Hash) bool {
		rec, e := rel.Create(h)
		if lasterr = e; e != nil {
			return false
		}

		if lasterr = fn(rec); lasterr != nil {
			return false
		}
		return true
	})

	if lasterr != nil {
		return lasterr
	}
	return err
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

func (rel *Relation) Find(id interface{}) (*ActiveRecord, error) {
	op := QueryOperation{
		TableName: rel.name + "s",
		Columns:   rel.scope.AttributeNames(),
		Values:    map[string]interface{}{rel.PrimaryKey(): id},
	}

	var rows []Hash

	if err := rel.conn.ExecQuery(rel.Context(), &op, func(h Hash) bool {
		rows = append(rows, h)
		return true
	}); err != nil {
		return nil, err
	}

	if len(rows) != 1 {
		return nil, &ErrRecordNotFound{PrimaryKey: rel.PrimaryKey(), ID: id}
	}
	return rel.Create(rows[0])
}

// ToA converts Relation to array. The method access database to retrieve objects.
func (rel *Relation) ToA() ([]*ActiveRecord, error) {
	var rr []*ActiveRecord

	if err := rel.Each(func(r *ActiveRecord) error {
		rr = append(rr, r)
		return nil
	}); err != nil {
		return nil, err
	}

	return rr, nil
}
