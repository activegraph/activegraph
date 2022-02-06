package activerecord

import (
	"context"
	"fmt"
	"strings"

	. "github.com/activegraph/activegraph/activesupport"
)

type ErrUnknownPrimaryKey struct {
	PrimaryKey  string
	Description string
}

func (e *ErrUnknownPrimaryKey) Error() string {
	return fmt.Sprintf("Primary key is unknown, %s", e.Description)
}

type R struct {
	rel *Relation

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

func (r *R) DefineAttribute(name string, t Type, validators ...AttributeValidator) {
	r.attrs[name] = attr{Name: name, Type: t}
	r.validators.include(name, typeValidator{t})
	r.validators.include(name, validators...)
}

func (r *R) Validates(name string, validator AttributeValidator) {
	if v, ok := validator.(Initializer); ok {
		err := v.Initialize()
		if err != nil {
			panic(err)
		}
		// Err(err).Unwrap()
	}
	r.validators.include(name, validator)
}

func (r *R) ValidatesPresence(names ...string) {
	r.validators.extend(names, new(Presence))
}

func (r *R) BelongsTo(name string, init ...func(*BelongsTo)) {
	assoc := BelongsTo{targetName: name, owner: r.rel, reflection: r.reflection}

	switch len(init) {
	case 0:
	case 1:
		init[0](&assoc)
	default:
		panic(ErrMultipleVariadicArguments{Name: "init"})
	}

	r.attrs[assoc.AssociationForeignKey()] = attr{
		Name: assoc.AssociationForeignKey(),
		Type: Nil{new(Int64)},
	}
	r.assocs[name] = &assoc
}

func (r *R) HasMany(name string) {
	// TODO: Define library methods to pluralize words.
	targetName := strings.TrimSuffix(name, "s")

	// Use plural name for the name of attribute, while target name
	// of the association should be in singular (to find a target relation
	// through the reflection.
	r.assocs[name] = &HasMany{
		targetName: targetName, owner: r.rel, reflection: r.reflection,
	}
}

func (r *R) HasOne(name string) {
	r.assocs[name] = &HasOne{targetName: name, owner: r.rel, reflection: r.reflection}
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
		columnType := column.Type
		if !column.NotNull {
			columnType = Nil{columnType}
		}
		r.DefineAttribute(column.Name, columnType)

		if column.IsPrimaryKey {
			r.PrimaryKey(column.Name)
		}
	}
	return nil
}

type Relation struct {
	name      string
	tableName string
	// TODO: add *Reflection property.
	// reflection *Reflection

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
		panic(&ErrMultipleVariadicArguments{Name: "init"})
	}

	if err != nil {
		panic(err)
	}
	return rel
}

func Initialize(name string, init func(*R)) (*Relation, error) {
	rel := &Relation{name: name}

	r := R{
		rel:         rel,
		assocs:      make(associationsMap),
		attrs:       make(attributesMap),
		validators:  make(validatorsMap),
		reflection:  globalReflection,
		connections: globalConnectionHandler,
	}

	err := r.init(context.TODO(), name+"s")
	if err != nil {
		return nil, err
	}
	if init != nil {
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
	rel.tableName = r.tableName
	rel.scope = scope
	rel.associations = *assocs
	rel.validations = *validations
	rel.connections = r.connections
	rel.query = &QueryBuilder{from: r.tableName}
	rel.AttributeMethods = scope
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

func (rel *Relation) New(params ...map[string]interface{}) RecordResult {
	switch len(params) {
	case 0:
		return ReturnRecord(rel.Initialize(nil))
	case 1:
		return ReturnRecord(rel.Initialize(params[0]))
	default:
		return ErrRecord(&ErrMultipleVariadicArguments{Name: "params"})
	}
}

func (rel *Relation) Initialize(params map[string]interface{}) (*ActiveRecord, error) {
	attributes := rel.scope.clear()
	err := attributes.AssignAttributes(params)
	if err != nil {
		return nil, err
	}

	rec := &ActiveRecord{
		name:         rel.name,
		tableName:    rel.tableName,
		conn:         rel.Connection(),
		attributes:   attributes,
		associations: rel.associations.copy(),
		validations:  *rel.validations.copy(),
	}
	return rec.init(), nil
}

func (rel *Relation) Create(params map[string]interface{}) RecordResult {
	return ReturnRecord(rel.Initialize(params)).Insert()
}

func (rel *Relation) ExtractRecord(h Hash) (*ActiveRecord, error) {
	var (
		attrNames   = rel.scope.AttributeNames()
		columnNames = rel.scope.ColumnNames()
	)

	params := make(Hash, len(attrNames))
	for i, colName := range columnNames {
		attrName := attrNames[i]
		attr := rel.scope.AttributeForInspect(attrName)

		attrValue, err := attr.AttributeType().Deserialize(h[colName])
		if err != nil {
			return nil, err
		}
		params[attrName] = attrValue
	}

	return rel.Initialize(params)
}

// PrimaryKey returns the attribute name of the record's primary key.
func (rel *Relation) PrimaryKey() string {
	return rel.scope.PrimaryKey()
}

func (rel *Relation) All() CollectionResult {
	return OkCollection(rel)
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

	err := rel.Connection().ExecQuery(rel.Context(), q.Operation(), func(h Hash) bool {
		rec, e := rel.ExtractRecord(h)
		if lasterr = e; e != nil {
			return false
		}

		for _, join := range rel.query.joinValues {
			arec, e := join.Relation.ExtractRecord(h)
			if lasterr = e; e != nil {
				return false
			}

			// TODO: Fix this assignment, it should return an error.
			rec.associations.set(join.Relation.Name(), arec)
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

func (rel *Relation) Find(id interface{}) RecordResult {
	var q QueryBuilder
	q.From(rel.TableName())
	q.Select(rel.scope.AttributeNames()...)
	// TODO: consider using unified approach.
	q.Where(fmt.Sprintf("%s = ?", rel.PrimaryKey()), id)

	var rows []Hash

	if err := rel.Connection().ExecQuery(rel.Context(), q.Operation(), func(h Hash) bool {
		rows = append(rows, h)
		return true
	}); err != nil {
		return ErrRecord(err)
	}

	if len(rows) != 1 {
		return ErrRecord(&ErrRecordNotFound{PrimaryKey: rel.PrimaryKey(), ID: id})
	}
	return rel.New(rows[0])
}

// FindBy returns a record matching the specified condition.
//
//	person := Person.FindBy("name", "Bill")
//	// Ok(Some(#<Person id: 1, name: "Bill", occupation: "retired">))
//
//	person := Person.FindBy("salary > ?", 10000)
//	// Ok(Some(#<Person id: 3, name: "Jeff", occupation: "CEO">))
func (rel *Relation) FindBy(cond string, arg interface{}) RecordResult {
	return rel.Where(cond, arg).First()
}

// First find returns the first record.
func (rel *Relation) First() RecordResult {
	records, err := rel.Limit(1).ToA()
	if err != nil {
		return ErrRecord(err)
	}
	switch len(records) {
	case 0:
		return OkRecord(nil)
	default:
		return OkRecord(records[0])
	}
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

func (rel *Relation) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%s(", strings.Title(rel.name))

	attrs := rel.AttributesForInspect()
	for i, attr := range attrs {
		fmt.Fprintf(&buf, "%s: %s", attr.AttributeName(), attr.AttributeType())
		if i < len(attrs)-1 {
			fmt.Fprint(&buf, ", ")
		}
	}

	fmt.Fprintf(&buf, ")")
	return buf.String()
}
