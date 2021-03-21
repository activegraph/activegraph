package activerecord

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	Int    = "int"
	String = "string"
)

type Attribute interface {
	Name() string
	CastType() string
	Validate(value interface{}) error
}

type RWAttr struct {
	name       string
	castType   string
	validators []func(interface{}) error
}

// Name returns the name of the attribute.
func (a *RWAttr) Name() string {
	return a.name
}

func (a *RWAttr) CastType() string {
	return a.castType
}

func (a *RWAttr) Ensure(func(interface{}) error) *RWAttr {
	return a
}

func (a *RWAttr) Validate(val interface{}) error {
	for i := 0; i < len(a.validators); i++ {
		if err := a.validators[i](val); err != nil {
			return err
		}
	}
	return nil
}

func Attr(name string, castType string) *RWAttr {
	return &RWAttr{name: name, castType: castType}
}

func MaxLen(num int) func(interface{}) error {
	if num < 0 {
		panic("num is less zero")
	}
	return func(val interface{}) error {
		s, ok := val.(string)
		if !ok {
			return errors.Errorf("%q is not a string", val)
		}
		if len(s) > num {
			return errors.Errorf("%q lenght is >%d", val, num)
		}
		return nil
	}
}

type AssocAttr struct {
	model   string
	through *string
}

func (a *AssocAttr) Name() string {
	return a.model
}

func (a *AssocAttr) CastType() string {
	return ""
}

func (a *AssocAttr) Validate(value interface{}) error {
	return nil
}

func (a *AssocAttr) Through(attr string) *AssocAttr {
	a.through = &attr
	return a
}

func BelongsTo(model string) *AssocAttr {
	return &AssocAttr{model: model}
}

type ErrUnknownAttribute struct {
	RecordName string
	Attr       string
}

func (e *ErrUnknownAttribute) Error() string {
	return fmt.Sprintf("unknown attribute %q for %s", e.Attr, e.RecordName)
}

type attributes struct {
	recordName string
	keys       map[string]Attribute
	values     map[string]interface{}
}

// newAttributes creates a new collection of attributes for the specified record.
func newAttributes(
	recordName string, attrs []Attribute, values map[string]interface{},
) attributes {

	keys := make(map[string]Attribute, len(attrs))
	for i := 0; i < len(attrs); i++ {
		keys[attrs[i].Name()] = attrs[i]
	}
	return attributes{recordName, keys, values}
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
