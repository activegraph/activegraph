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

func (r *ActiveRecord) Move(dst interface{}) error {
	return nil
}

func (r *ActiveRecord) Borrow(src interface{}) error {
	return nil
}

func (r *ActiveRecord) Insert(ctx context.Context) (*ActiveRecord, error) {
	op := InsertOperation{
		// TODO: specify plural name of a record table.
		TableName: r.recordName + "s",
		Values:    r.values,
	}

	err := r.conn.ExecInsert(ctx, r.conn.BuildInsertStmt(&op))
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *ActiveRecord) Update(ctx context.Context) (*ActiveRecord, error) {
	return nil, nil
}

func (r *ActiveRecord) Delete(ctx context.Context) error {
	return nil
}

func (r *ActiveRecord) IsPersisted() bool {
	return false
}

type Schema struct {
	name  string
	conn  Conn
	attrs []Attribute
}

func New(name string, attrs ...Attribute) *Schema {
	return &Schema{name: name, attrs: attrs}
}

func (r *Schema) Connect(conn Conn) *Schema {
	r.conn = conn
	return r
}

func (r *Schema) New(params map[string]interface{}) *ActiveRecord {
	return &ActiveRecord{
		new:        true,
		conn:       r.conn,
		attributes: newAttributes(r.name, r.attrs, params),
	}
}
