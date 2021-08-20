package activerecord

import (
	"context"
	"fmt"

	"github.com/activegraph/activegraph/activesupport"
)

type ErrUnknownPrimaryKey struct {
	PrimaryKey  string
	Description string
}

func (e *ErrUnknownPrimaryKey) Error() string {
	return fmt.Sprintf("Primary key is unknown, %s", e.Description)
}

type R struct {
	tableName   string
	primaryKey  string
	attrs       attributesMap
	assocs      associationsMap
	validators  validatorsMap
	reflection  *Reflection
	connections *connectionHandler
}

// TableName sets the table name explicitly.
//
//	Vertex := activerecord.New("vertex", func(r *activerecord.R) {
//		r.TableName("vertices")
//	})
func (r *R) TableName(name string) {
	r.tableName = name
}

func (r *R) PrimaryKey(name string) {
	r.primaryKey = name
}

func (r *R) AttrInt(name string) {
	r.attrs[name] = IntAttr{Name: name}
	r.validators.include(name, new(IntValidator))
}

func (r *R) AttrString(name string) {
	r.attrs[name] = StringAttr{Name: name}
	r.validators.include(name, new(StringValidator))
}

func (r *R) AttrFloat(name string) {
	r.attrs[name] = FloatAttr{Name: name}
	r.validators.include(name, new(FloatValidator))
}

func (r *R) AttrBoolean(name string) {
	r.attrs[name] = BooleanAttr{Name: name}
	r.validators.include(name, new(BooleanValidator))
}

func (r *R) Validates(name string, validator AttributeValidator) {
	if v, ok := validator.(activesupport.Initializer); ok {
		err := v.Initialize()
		activesupport.Err(err).Unwrap()
	}
	r.validators.include(name, validator)
}

func (r *R) Scope(reflection *Reflection) {
	if reflection == nil {
		panic("nil reflection")
	}
	r.reflection = reflection
}

func (r *R) BelongsTo(name string, init ...func(*BelongsTo)) {
	assoc := BelongsTo{name: name}

	switch len(init) {
	case 0:
	case 1:
		init[0](&assoc)
	default:
		panic("multiple initializations passed")
	}

	r.attrs[assoc.AssociationForeignKey()] = IntAttr{
		Name: assoc.AssociationForeignKey(),
	}
	r.assocs[name] = &assoc
}

func (r *R) HasMany(name string) {
	r.assocs[name] = &HasMany{name: name}
}

func (r *R) init(ctx context.Context, tableName string) error {
	conn, err := r.connections.RetrieveConnection(primaryConnectionName)
	if err != nil {
		return err
	}

	definitions, err := conn.ColumnDefinitions(ctx, tableName)
	if err != nil {
		return err
	}

	for _, column := range definitions {
		switch column.Type {
		case "integer":
			r.AttrInt(column.Name)
		case "varchar":
			r.AttrString(column.Name)
		case "float":
			r.AttrFloat(column.Name)
		case "boolean":
			r.AttrBoolean(column.Name)
		}

		if column.IsPrimaryKey {
			r.PrimaryKey(column.Name)
		}
	}
	return nil
}

type Relation struct {
	name      string
	tableName string

	conn        Conn
	connections *connectionHandler

	scope *attributes
	query *QueryBuilder
	ctx   context.Context

	associations
	validations
	AttributeMethods
}

func New(name string, init ...func(*R)) *Relation {
	var (
		rel *Relation
		err error
	)
	switch len(init) {
	case 0:
		rel, err = Initialize(name, nil)
	case 1:
		rel, err = Initialize(name, init[0])
	default:
		panic(&activesupport.ErrMultipleVariadicArguments{Name: "init"})
	}

	if err != nil {
		panic(err)
	}
	return rel
}

func Initialize(name string, init func(*R)) (*Relation, error) {
	r := R{
		assocs:      make(associationsMap),
		attrs:       make(attributesMap),
		validators:  make(validatorsMap),
		reflection:  globalReflection,
		connections: globalConnectionHandler,
	}

	if init == nil {
		r.init(context.TODO(), name+"s")
	} else {
		init(&r)
	}

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
	if r.tableName == "" {
		r.tableName = name + "s"
	}

	// The scope is empty by default.
	scope, err := newAttributes(name, r.attrs.copy(), nil)
	if err != nil {
		return nil, err
	}

	assocs := newAssociations(name, r.assocs.copy(), r.reflection)
	validations := newValidations(r.validators.copy())

	// Create the model schema, and register it within a reflection instance.
	rel := &Relation{
		name:             name,
		tableName:        r.tableName,
		scope:            scope,
		associations:     *assocs,
		validations:      *validations,
		connections:      r.connections,
		query:            &QueryBuilder{from: r.tableName},
		AttributeMethods: scope,
	}
	r.reflection.AddReflection(name, rel)

	return rel, nil
}

func (rel *Relation) TableName() string {
	return rel.tableName
}

func (rel *Relation) Name() string {
	return rel.name
}

func (rel *Relation) Copy() *Relation {
	scope := rel.scope.copy()

	return &Relation{
		name:             rel.name,
		tableName:        rel.tableName,
		conn:             rel.Connection(),
		connections:      rel.connections,
		scope:            rel.scope.copy(),
		query:            rel.query.copy(),
		ctx:              rel.ctx,
		associations:     *rel.associations.copy(),
		validations:      *rel.validations.copy(),
		AttributeMethods: scope,
	}
}

func (rel *Relation) empty() *Relation {
	rel.scope, _ = newAttributes(rel.name, nil, nil)
	return rel
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
	newrel := rel.Copy()
	newrel.ctx = ctx
	return newrel
}

func (rel *Relation) Connect(conn Conn) *Relation {
	newrel := rel.Copy()
	newrel.conn = conn
	return newrel
}

func (rel *Relation) Connection() Conn {
	if rel.conn != nil {
		return rel.conn
	}

	conn, err := rel.connections.RetrieveConnection(primaryConnectionName)
	if err != nil {
		return &errConn{err}
	}
	return conn
}

func (rel *Relation) New(params ...map[string]interface{}) Result {
	switch len(params) {
	case 0:
		return Return(rel.Initialize(nil))
	case 1:
		return Return(rel.Initialize(params[0]))
	default:
		return Err(&activesupport.ErrMultipleVariadicArguments{Name: "params"})
	}
}

func (rel *Relation) Initialize(params map[string]interface{}) (*ActiveRecord, error) {
	attributes := rel.scope.clear()
	err := attributes.AssignAttributes(params)
	if err != nil {
		return nil, err
	}

	return &ActiveRecord{
		name:               rel.name,
		tableName:          rel.tableName,
		conn:               rel.Connection(),
		attributes:         *attributes,
		associations:       *rel.associations.copy(),
		validations:        *rel.validations.copy(),
		associationRecords: make(map[string]*ActiveRecord),
	}, nil
}

func (rel *Relation) Create(params map[string]interface{}) (*ActiveRecord, error) {
	rec, err := rel.Initialize(params)
	if err != nil {
		return nil, err
	}
	return rec.Insert()
}

func (rel *Relation) ExtractRecord(h activesupport.Hash) (*ActiveRecord, error) {
	var (
		attrNames   = rel.scope.AttributeNames()
		columnNames = rel.scope.ColumnNames()
	)

	params := make(map[string]interface{}, len(attrNames))
	for i, colName := range columnNames {
		params[attrNames[i]] = h[colName]
	}

	return rel.Initialize(params)
}

// PrimaryKey returns the attribute name of the record's primary key.
func (rel *Relation) PrimaryKey() string {
	return rel.scope.PrimaryKey()
}

func (rel *Relation) All() *Relation {
	return rel.Copy()
}

// TODO: move to the Schema type all column-related methods.
func (rel *Relation) ColumnNames() []string {
	return rel.scope.ColumnNames()
}

func (rel *Relation) Each(fn func(*ActiveRecord) error) error {
	q := rel.query.copy()
	q.Select(rel.ColumnNames()...)

	// Include all join dependencies into the query with fully-qualified column
	// names, so each part of the request can be extracted individually.
	for _, join := range rel.query.joinValues {
		q.Select(join.Relation.ColumnNames()...)
	}

	var lasterr error

	err := rel.Connection().ExecQuery(rel.Context(), q.Operation(), func(h activesupport.Hash) bool {
		rec, e := rel.ExtractRecord(h)
		if lasterr = e; e != nil {
			return false
		}

		for _, join := range rel.query.joinValues {
			arec, e := join.Relation.ExtractRecord(h)
			if lasterr = e; e != nil {
				return false
			}

			e = rec.AssignAssociation(join.Relation.Name(), arec)
			if lasterr = e; e != nil {
				return false
			}
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
		// newrel.scope.AssignAttribute(cond, arg)
		newrel.query.Where(fmt.Sprintf("%s = ?", cond), arg)
	} else {
		newrel.query.Where(cond, arg)
	}
	return newrel
}

// Select allows to specify a subset of fields to return.
//
// Method returns a new relation, where a set of attributes is limited by the
// specified list.
//
//	Model.Select("field", "other_field")
//	// #<Model id: 1, field: "value", other_field: "value">
//
// Accessing attributes of a Record that do not have fields retrieved by a select
// except id with return nil.
//
//	model, _ := Model.Select("field").Find(1)
//	model.Attribute("other_field") // Returns nil
func (rel *Relation) Select(attrNames ...string) *Relation {
	newrel := rel.Copy()

	if !newrel.scope.HasAttributes(attrNames...) {
		return newrel.empty()
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
		return newrel.empty()
	}

	newrel.query.Group(attrNames...)
	return newrel
}

// Limit specifies a limit for the number of records to retrieve.
//
//	User.Limit(10) // Generated SQL has 'LIMIT 10'
func (rel *Relation) Limit(num int) *Relation {
	newrel := rel.Copy()
	newrel.query.Limit(num)
	return newrel
}

func (rel *Relation) Joins(assocNames ...string) *Relation {
	newrel := rel.Copy()

	for _, assocName := range assocNames {
		association := newrel.ReflectOnAssociation(assocName)
		if association == nil {
			return newrel.empty()
		}

		newrel.query.Join(association.Relation.Copy(), association.Association)
	}
	return newrel
}

func (rel *Relation) Find(id interface{}) Result {
	var q QueryBuilder
	q.From(rel.TableName())
	q.Select(rel.scope.AttributeNames()...)
	// TODO: consider using unified approach.
	q.Where(fmt.Sprintf("%s = ?", rel.PrimaryKey()), id)

	var rows []activesupport.Hash

	if err := rel.Connection().ExecQuery(rel.Context(), q.Operation(), func(h activesupport.Hash) bool {
		rows = append(rows, h)
		return true
	}); err != nil {
		return Err(err)
	}

	if len(rows) != 1 {
		return Err(ErrRecordNotFound{PrimaryKey: rel.PrimaryKey(), ID: id})
	}
	return rel.New(rows[0])
}

func (rel *Relation) InsertAll(params ...map[string]interface{}) (
	rr []*ActiveRecord, err error,
) {
	rr = make([]*ActiveRecord, 0, len(params))
	for _, h := range params {
		rec, err := rel.Initialize(h)
		if err != nil {
			return nil, err
		}

		rr = append(rr, rec)
	}

	if err = rel.connections.Transaction(rel.Context(), func() error {
		for i, rec := range rr {
			if rr[i], err = rec.Insert(); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return rr, nil
}

// ToA converts Relation to array. The method access database to retrieve objects.
func (rel *Relation) ToA() (Array, error) {
	var rr Array

	if err := rel.Each(func(r *ActiveRecord) error {
		rr = append(rr, r)
		return nil
	}); err != nil {
		return nil, err
	}

	return rr, nil
}

// ToSQL returns sql statement for the relation.
//
//	User.Where("name", "Oscar").ToSQL()
//	// SELECT * FROM "users" WHERE "name" = ?
func (rel *Relation) ToSQL() string {
	return rel.query.String()
}
