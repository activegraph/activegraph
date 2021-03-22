package activerecord

import (
	"fmt"
)

const (
	Int    = "int"
	String = "string"
)

type Attribute interface {
	AttributeName() string
	IsPrimaryKey() bool
	CastType() string

	Validator
}

type IntAttr struct {
	Name       string
	PrimaryKey bool
	Validates  IntValidators
}

func (a IntAttr) AttributeName() string            { return a.Name }
func (a IntAttr) IsPrimaryKey() bool               { return a.PrimaryKey }
func (a IntAttr) CastType() string                 { return Int }
func (a IntAttr) Validate(value interface{}) error { return a.Validates.Validate(value) }

type StringAttr struct {
	Name       string
	PrimaryKey bool
	Validates  StringValidators
}

func (a StringAttr) AttributeName() string            { return a.Name }
func (a StringAttr) IsPrimaryKey() bool               { return a.PrimaryKey }
func (a StringAttr) CastType() string                 { return String }
func (a StringAttr) Validate(value interface{}) error { return a.Validates.Validate(value) }

// ErrUnknownAttribute is returned on attempt to assign unknown attribute to the
// ActiveRecord.
type ErrUnknownAttribute struct {
	RecordName string
	Attr       string
}

// Error returns a string representation of the error.
func (e *ErrUnknownAttribute) Error() string {
	return fmt.Sprintf("unknown attribute %q for %s", e.Attr, e.RecordName)
}

const (
	// default name of the primary key.
	defaultPrimaryKeyName = "id"
)

// attributes of the ActiveRecord.
type attributes struct {
	recordName string
	primaryKey Attribute
	keys       map[string]Attribute
	values     map[string]interface{}
}

// newAttributes creates a new collection of attributes for the specified record.
func newAttributes(
	recordName string, attrs []Attribute, values map[string]interface{},
) attributes {

	var (
		primaryKey Attribute
		keys       = make(map[string]Attribute, len(attrs))
	)
	for i := 0; i < len(attrs); i++ {
		keys[attrs[i].AttributeName()] = attrs[i]

		if attrs[i].IsPrimaryKey() {
			if primaryKey != nil {
				panic("multiple primary keys are not allowed")
			}
			primaryKey = attrs[i]
		}
	}

	// When the primary key attribute was not specified directly, generate
	// a new "id" integer attribute, ensure that the attribute with the same
	// name is not presented in the schema definition.
	if _, dup := keys[defaultPrimaryKeyName]; dup {
		panic(fmt.Sprintf("%q is an attribute, but not a primary key", defaultPrimaryKeyName))
	}
	if primaryKey == nil {
		primaryKey = IntAttr{Name: defaultPrimaryKeyName, PrimaryKey: true}
		keys[defaultPrimaryKeyName] = primaryKey
	}

	return attributes{recordName, primaryKey, keys, values}
}

// ID returns the primary key column's value.
func (a *attributes) ID() interface{} {
	return a.values[a.primaryKey.AttributeName()]
}

// AttributeNames return an array of names for the attributes available on this object.
func (a *attributes) AttributeNames() []string {
	names := make([]string, len(a.keys))
	for name := range a.keys {
		names = append(names, name)
	}
	return names
}

// HasAttribute returns true if the given table attribute is in the attribute map,
// otherwise false.
func (a *attributes) HasAttribute(attrName string) bool {
	_, ok := a.keys[attrName]
	return ok
}

// AssignAttribute allows to set attribute by the name.
//
// Method return an error when value does not pass validation of the attribute.
func (a *attributes) AssignAttribute(attrName string, val interface{}) error {
	attr, ok := a.keys[attrName]
	if !ok {
		return &ErrUnknownAttribute{RecordName: a.recordName, Attr: attrName}
	}
	// Ensure that attribute passes validation.
	if err := attr.Validate(val); err != nil {
		return err
	}

	if a.values == nil {
		a.values = make(map[string]interface{})
	}
	a.values[attrName] = val
	return nil
}

// AccessAttribute returns the value of the attribute identified by attrName.
func (a *attributes) AccessAttribute(attrName string) (val interface{}) {
	if !a.HasAttribute(attrName) {
		return nil
	}
	return a.values[attrName]
}

// AttributePresent returns true if the specified attribute has been set by the user
// or by a database and is not nil, otherwise false.
func (a *attributes) AttributePresent(attrName string) bool {
	if _, ok := a.keys[attrName]; !ok {
		return false
	}
	return a.values[attrName] != nil
}
