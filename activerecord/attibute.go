package activerecord

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/activegraph/activegraph/activesupport"
)

// primaryKey must implement attributes that are primary keys.
type primaryKey interface {
	PrimaryKey() bool
}

type Attribute interface {
	AttributeName() string
	AttributeType() Type
}

type AttributeMethods interface {
	AttributeNames() []string
	HasAttribute(attrName string) bool
	HasAttributes(attrNames ...string) bool
	AttributeForInspect(attrName string) Attribute
	AttributesForInspect(attrNames ...string) []Attribute
}

type AttributeAccessors interface {
	ID() interface{}
	AttributePresent(attrName string) bool
	Attribute(attrName string) interface{}
	// AccessAttribute(attrName string) interface{}
	AssignAttribute(attrName string, val interface{}) error
	AssignAttributes(newAttributes map[string]interface{}) error
}

// PrimaryKey makes any specified attribute a primary key.
type PrimaryKey struct {
	Attribute
}

// PrimaryKey always returns true.
func (p PrimaryKey) PrimaryKey() bool {
	return true
}

type attr struct {
	Name string
	Type Type
}

func (a attr) AttributeName() string {
	return a.Name
}

func (a attr) AttributeType() Type {
	return a.Type
}

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

type attributesMap map[string]Attribute

func (m attributesMap) copy() attributesMap {
	mm := make(attributesMap, len(m))
	for name, attr := range m {
		mm[name] = attr
	}
	return mm
}

// attributes of the ActiveRecord.
type attributes struct {
	recordName string
	primaryKey Attribute
	keys       attributesMap
	values     activesupport.Hash
}

func (a *attributes) copy() *attributes {
	return &attributes{
		recordName: a.recordName,
		primaryKey: a.primaryKey,
		keys:       a.keys.copy(),
		values:     a.values.Copy(),
	}
}

func (a *attributes) clear() *attributes {
	newa := a.copy()
	newa.values = make(activesupport.Hash, len(a.keys))
	return newa
}

func (a *attributes) merge(a1 *attributes) *attributes {
	for attrName, attr := range a1.keys {
		a.keys[attrName] = attr
	}
	for attrName, attrVal := range a1.values {
		a.values[attrName] = attrVal
	}
	return a
}

// newAttributes creates a new collection of attributes for the specified record.
func newAttributes(recordName string, attrs attributesMap, values activesupport.Hash) (
	*attributes, error,
) {

	recordAttrs := attributes{
		recordName: recordName,
		keys:       attrs,
		values:     values,
	}
	for _, attr := range recordAttrs.keys {
		// Save the primary key attribute as a standalone property for
		// easier access to it.
		if pk, ok := attr.(primaryKey); ok && pk.PrimaryKey() {
			if recordAttrs.primaryKey != nil {
				return nil, errors.New("multiple primary keys are not supported")
			}
			recordAttrs.primaryKey = attr
		}
	}

	// When the primary key attribute was not specified directly, generate
	// a new "id" integer attribute, ensure that the attribute with the same
	// name is not presented in the schema definition.
	if _, dup := recordAttrs.keys[defaultPrimaryKeyName]; dup && recordAttrs.primaryKey == nil {
		err := errors.Errorf("%q is an attribute, but not a primary key", defaultPrimaryKeyName)
		return nil, err
	}
	if recordAttrs.primaryKey == nil {
		pk := PrimaryKey{Attribute: attr{Name: defaultPrimaryKeyName, Type: new(Int64)}}
		recordAttrs.primaryKey = pk

		if recordAttrs.keys == nil {
			recordAttrs.keys = make(attributesMap)
		}
		recordAttrs.keys[defaultPrimaryKeyName] = pk
	}

	// Enforce values within a record, all of them must be
	// presented in the specified list of attributes.
	for attrName := range recordAttrs.values {
		if _, ok := recordAttrs.keys[attrName]; !ok {

			err := &ErrUnknownAttribute{RecordName: recordName, Attr: attrName}
			return nil, err
		}
	}

	return &recordAttrs, nil
}

func (a *attributes) each(fn func(name string, value interface{})) {
	for attrName, value := range a.values {
		fn(attrName, value)
	}
}

func (a *attributes) PrimaryKey() string {
	return a.primaryKey.AttributeName()
}

// ID returns the primary key column's value.
func (a *attributes) ID() interface{} {
	return a.values[a.primaryKey.AttributeName()]
}

// AttributeNames return an array of names for the attributes available on this object.
func (a *attributes) AttributeNames() []string {
	names := make([]string, 0, len(a.keys))
	for name := range a.keys {
		names = append(names, name)
	}
	sort.StringSlice(names).Sort()
	return names
}

func (a *attributes) ColumnNames() []string {
	names := make([]string, 0, len(a.keys))
	for name := range a.keys {
		names = append(names, a.recordName+"s."+name)
	}
	sort.StringSlice(names).Sort()
	return names
}

// HasAttribute returns true if the given table attribute is in the attribute map,
// otherwise false.
func (a *attributes) HasAttribute(attrName string) bool {
	_, ok := a.keys[attrName]
	return ok
}

func (a *attributes) HasAttributes(attrNames ...string) bool {
	for _, attrName := range attrNames {
		if !a.HasAttribute(attrName) {
			return false
		}
	}
	return true
}

// AssignAttribute allows to set attribute by the name.
//
// Method return an error when value does not pass validation of the attribute.
func (a *attributes) AssignAttribute(attrName string, val interface{}) error {
	if !a.HasAttribute(attrName) {
		return &ErrUnknownAttribute{RecordName: a.recordName, Attr: attrName}
	}
	// TODO: Ensure that attribute passes validation?
	// if err := attr.Validate(val); err != nil {
	// 	return err
	// }

	if a.values == nil {
		a.values = make(activesupport.Hash)
	}
	a.values[attrName] = val
	return nil
}

// AssignAttributes allows to set all the attributes by passing in a map of attributes
// with keys matching attributet names.
//
// The method either assigns all provided attributes, no attributes are assigned
// in case of error.
func (a *attributes) AssignAttributes(newAttributes map[string]interface{}) error {
	// Create a copy of attributes, either update all attributes or
	// return the object in the previous state.
	var (
		keys   = a.keys.copy()
		values = a.values.Copy()
	)

	for attrName, val := range newAttributes {
		err := a.AssignAttribute(attrName, val)
		if err != nil {
			// Return the original state of the attributes.
			a.keys = keys
			a.values = values
			return err
		}
	}
	return nil
}

// AccessAttribute returns the value of the attribute identified by attrName.
func (a *attributes) AccessAttribute(attrName string) (val interface{}) {
	if !a.HasAttribute(attrName) {
		return nil
	}
	return a.values[attrName]
}

// Attribute is an alias for AccessAttribute.
func (a *attributes) Attribute(attrName string) (val interface{}) {
	return a.AccessAttribute(attrName)
}

// AttributePresent returns true if the specified attribute has been set by the user
// or by a database and is not nil, otherwise false.
func (a *attributes) AttributePresent(attrName string) bool {
	if !a.HasAttribute(attrName) {
		return false
	}
	return a.values[attrName] != nil
}

func (a *attributes) AttributeForInspect(attrName string) Attribute {
	if !a.HasAttribute(attrName) {
		return nil
	}
	return a.keys[attrName]
}

func (a *attributes) AttributesForInspect(attrNames ...string) []Attribute {
	attrs := make([]Attribute, 0, len(attrNames))
	if len(attrNames) == 0 {
		attrNames = a.AttributeNames()
	}
	for _, attrName := range attrNames {
		if a.HasAttribute(attrName) {
			attrs = append(attrs, a.keys[attrName])
		}
	}
	return attrs
}

// ExceptAttribute removes the specified attribute. Method returns error when attribute
// is unknown.
func (a *attributes) ExceptAttribute(attrName string) error {
	if !a.HasAttribute(attrName) {
		return &ErrUnknownAttribute{RecordName: a.recordName, Attr: attrName}
	}
	delete(a.keys, attrName)
	delete(a.values, attrName)
	return nil
}
