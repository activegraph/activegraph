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
	tableName  string
	conn       Conn
	ctx        context.Context
	reflection *Reflection

	attributes
	associations
}

func (r *ActiveRecord) Copy() *ActiveRecord {
	return &ActiveRecord{
		name:       r.name,
		tableName:  r.tableName,
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
	for attrName, attr := range r.attributes.keys {
		if err := attr.Validate(r.attributes.values[attrName]); err != nil {
			return err
		}
	}
	return nil
}

func (r *ActiveRecord) AccessAssociation(assocName string) (*ActiveRecord, error) {
	assocRec := r.associations.AccessAssociation(assocName)
	if assocRec != nil {
		return assocRec, nil
	}

	assocId := r.AccessAttribute(r.associations.get(assocName).AssociationForeignKey())
	assocRel, err := r.reflection.Reflection(assocName)
	if err != nil {
		return nil, err
	}

	return assocRel.WithContext(r.Context()).Find(assocId)
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

// Collection returns a Relation of all associated records. A nil is returned
// if relation does not belong to the record.
func (r *ActiveRecord) Collection(assocName string) *Relation {
	rel, _ := r.AccessCollection(assocName)
	return rel
}

func (r *ActiveRecord) Insert() (*ActiveRecord, error) {
	op := InsertOperation{
		TableName: r.tableName,
		Values:    r.attributes.values,
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
		TableName:  r.tableName,
		PrimaryKey: r.primaryKey.AttributeName(),
		Value:      r.ID(),
	}

	return r.conn.ExecDelete(r.Context(), &op)
}

func (r *ActiveRecord) IsPersisted() bool {
	return false
}
