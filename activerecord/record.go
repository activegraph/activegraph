package activerecord

import (
	"context"
)

type ActiveRecord struct {
	new  bool
	conn Conn

	attributes
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
		Value:      r.values[r.primaryKey.AttributeName()],
	}

	return r.conn.ExecDelete(ctx, &op)
}

func (r *ActiveRecord) IsPersisted() bool {
	return false
}

type R struct {
	primaryKey string
	attrs      []Attribute
}

func (r *R) PrimaryKey(name string) {
	r.primaryKey = name
}

func (r *R) AttrInt(name string, validates ...IntValidator) {
	r.attrs = append(r.attrs, IntAttr{Name: name, Validates: validates})
}

func (r *R) AttrString(name string, validates ...StringValidator) {
	r.attrs = append(r.attrs, StringAttr{Name: name, Validates: validates})
}

func (d *R) BelongsTo(name string) {
}

func (d *R) HasMany(name string) {
}

type ModelSchema struct {
	name  string
	conn  Conn
	attrs []Attribute
}

func New(name string, defineRecord func(*R)) *ModelSchema {
	var r R
	defineRecord(&r)

	return &ModelSchema{name: name, attrs: r.attrs}
}

// PrimaryKey returns the attribute name of the record's primary key.
func (r *ModelSchema) PrimaryKey() string {
	attrs := newAttributes(r.name, r.attrs, nil)
	return attrs.primaryKey.AttributeName()
}

func (r *ModelSchema) Connect(conn Conn) *ModelSchema {
	r.conn = conn
	return r
}

func (r *ModelSchema) New(params map[string]interface{}) *ActiveRecord {
	return &ActiveRecord{
		new:        true,
		conn:       r.conn,
		attributes: newAttributes(r.name, r.attrs, params),
	}
}

func (r *ModelSchema) All(ctx context.Context) ([]ActiveRecord, error) {
	return nil, nil
}
