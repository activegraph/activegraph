package activerecord

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/activegraph/activegraph/activesupport"
)

type ErrValidation struct {
	Model  *ActiveRecord
	Errors Errors
}

func (e ErrValidation) Error() string {
	errors := strings.Join(e.Errors.FullMessages(), ", ")
	return fmt.Sprintf("'%s' invalid%s", e.Model.Name(), errors)
}

type ErrInvalidValue struct {
	AttrName string
	Message  string
	Value    interface{}
}

func (e ErrInvalidValue) Error() string {
	if len(e.Message) != 0 {
		return fmt.Sprintf("'%s' %s", e.AttrName, e.Message)
	}
	return fmt.Sprintf("'%s' has invalid value '%v'", e.AttrName, e.Value)
}

type ErrInvalidType struct {
	AttrName string
	TypeName string
	Value    interface{}
}

func (e ErrInvalidType) Error() string {
	return fmt.Sprintf(
		"invalid value '%v' for %s type of attribute '%s'",
		e.Value, e.TypeName, e.AttrName,
	)
}

type Validator interface {
	Validate(r *ActiveRecord) error
}

type AttributeValidator interface {
	ValidateAttribute(r *ActiveRecord, attrName string, value interface{}) error
	AllowsNil() bool
	AllowsBlank() bool
}

type validatorsMap map[string][]AttributeValidator

func (m validatorsMap) copy() validatorsMap {
	mm := make(validatorsMap, len(m))
	for name, validators := range m {
		mm[name] = validators
	}
	return mm
}

func (m validatorsMap) include(attrName string, validators ...AttributeValidator) {
	attrValidators := m[attrName]
	m[attrName] = append(attrValidators, validators...)
}

func (m validatorsMap) extend(attrNames []string, validator AttributeValidator) {
	for _, attrName := range attrNames {
		m.include(attrName, validator)
	}
}

type Errors struct {
	errors map[string][]error
}

func (e *Errors) IsEmpty() bool {
	return len(e.errors) == 0
}

func (e *Errors) Add(key string, err error) {
	if e.errors == nil {
		e.errors = make(map[string][]error)
	}
	errors := e.errors[key]
	e.errors[key] = append(errors, err)
}

func (e *Errors) Delete(keys ...string) {
	if len(keys) == 0 {
		e.errors = make(map[string][]error)
		return
	}
	for _, key := range keys {
		delete(e.errors, key)
	}
}

func (e *Errors) FullMessages() []string {
	messages := make([]string, len(e.errors))
	for _, keyErrors := range e.errors {
		for _, err := range keyErrors {
			messages = append(messages, err.Error())
		}
	}
	return messages
}

type validations struct {
	validators validatorsMap
	errors     Errors
}

func newValidations(validators validatorsMap) *validations {
	return &validations{validators: validators.copy()}
}

func (v *validations) copy() *validations {
	return newValidations(v.validators)
}

func (v *validations) validate(rec *ActiveRecord) error {
	v.errors.Delete()

	for attrName, validators := range v.validators {
		value := rec.Attribute(attrName)

		for _, validator := range validators {
			if (value == nil && validator.AllowsNil()) ||
				(activesupport.IsBlank(value) && validator.AllowsBlank()) {
				continue
			}
			err := validator.ValidateAttribute(rec, attrName, value)
			if err != nil {
				v.errors.Add(attrName, err)
			}
		}
	}

	if !v.errors.IsEmpty() {
		return ErrValidation{Model: rec, Errors: v.errors}
	}
	return nil
}

func (v *validations) Errors() Errors {
	return v.errors
}

type typeValidator struct {
	Type
}

func (v typeValidator) AllowsNil() bool   { return true }
func (v typeValidator) AllowsBlank() bool { return true }

func (v typeValidator) ValidateAttribute(rec *ActiveRecord, attrName string, val interface{}) error {
	if _, err := v.Deserialize(val); err != nil {
		return ErrInvalidType{AttrName: attrName, TypeName: v.String(), Value: val}
	}
	return nil
}

// Presence validates that specified value of the attribute is not blank (as defined
// by activesupport.IsBlank).
//
//	Supplier := activerecord.New("supplier", func(r *activerecord.R) {
//		r.HasOne("account")
//		r.Validates("account", &activegraph.Presence{})
//	})
//
// The account attribute must be in the object and it cannot be blank.
type Presence struct {
	AllowNil bool

	// Message is a custom error message (default is "can't be blank").
	Message string
}

// AllowsNil returns true if nil values are allowed, and false otherwise.
func (p *Presence) AllowsNil() bool { return p.AllowNil }

// AllowsBlank returns false, since blank values are not allowed for this validator.
func (p *Presence) AllowsBlank() bool { return false }

// ValidateAttribute returns ErrInvalidValue when specified value is blank.
func (p *Presence) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	if activesupport.IsBlank(val) {
		message := activesupport.Strings(p.Message, "can't be blank").Find(activesupport.String.IsNotEmpty)
		return ErrInvalidValue{AttrName: attrName, Value: val, Message: string(message)}
	}
	return nil
}

// Format validates that specified value of the attribute is of the correct form, going
// by the regular expression provided.
//
// Use `With` property to require that the attribute matches the regular expression:
//
//	Supplier := activerecord.New("supplier", func(r *activerecord.R) {
//		r.Validates("name", &activerecord.Format{With: `[a-z]+`})
//	})
//
// Use `Without` property to require that the attribute does not match the regular
// epxression:
//
//	Supplier := activerecord.New("supplier", func(r *activerecord.R) {
//		r.Validates("phone", &activerecord.Format{Without: `[a-zA-Z]`})
//	})
//
// You must pass either `With` or `Without` as parameters. When both are empty strings
// or both expressions are specified, an error ErrArgument is returned.
type Format struct {
	With    activesupport.String
	Without activesupport.String

	AllowNil   bool
	AllowBlank bool

	// Message is a customer error message (default is "has invalid format").
	Message string

	re *regexp.Regexp
}

// AllowsNil returns true when nil values are allowed, and false otherwise.
func (f *Format) AllowsNil() bool { return f.AllowNil }

// AllowsBlank returns true when blank values are allowed, and false otherwise.
func (f *Format) AllowsBlank() bool { return f.AllowBlank }

// Initialize ensures the correctness of the parameters and compiles the given regular
// expression. When parameters are not valid, or regular expression is not compilable,
// method returns an error.
func (f *Format) Initialize() (err error) {
	if (f.With.IsEmpty() && f.Without.IsEmpty()) || (!f.With.IsEmpty() && !f.Without.IsEmpty()) {
		return activesupport.ErrArgument{
			Message: "format: either 'With' or 'Without' must be supplied (but not both)",
		}
	}

	re := f.With
	if !f.Without.IsEmpty() {
		re = f.Without
	}

	f.re, err = regexp.Compile(string(re))
	return err
}

// ValidateAttribute validates that the given value is in the required format.
//
// When value type is not a string, method returns ErrInvlidType error. In case of
// failed validation, method returns ErrInvalidValue error.
func (f *Format) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	s, ok := val.(string)
	if !ok {
		return ErrInvalidType{AttrName: attrName, TypeName: "String", Value: val}
	}

	match := f.re.Match([]byte(s))
	if (match && !f.Without.IsEmpty()) || (!match && f.Without.IsEmpty()) {
		message := activesupport.Strings(f.Message, "has invalid format").Find(activesupport.String.IsNotEmpty)
		return ErrInvalidValue{AttrName: attrName, Value: val, Message: string(message)}
	}
	return nil
}

// Inclusion validates that the specified value of the attribute is available in a slice.
//
//	Supplier := activerecord.New("supplier", func(r *activerecord.R) {
//		r.Validates("state", &activerecord.Inclusion{In: activesupport.Strings("NY", "MA")})
//	})
type Inclusion struct {
	In activesupport.Slice

	AllowNil   bool
	AllowBlank bool
}

// AllowsNil returnd true when nil values are allowed, and false otherwise.
func (i *Inclusion) AllowsNil() bool { return i.AllowNil }

// AllowsBlank returns true when blank values are allowed, and false otherwise.
func (i *Inclusion) AllowsBlank() bool { return i.AllowBlank }

// ValidateAttribute validates that the given value is in the provided slice. When it's
// not, method returns ErrInvalidValue error.
func (i *Inclusion) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	if !i.In.Contains(val) {
		return ErrInvalidValue{
			AttrName: attrName, Value: val, Message: "is not included in the list",
		}
	}
	return nil
}

// Exclusion validates that specified value of the attribute is not in a slice.
//
//	Supplier := activerecord.New("supplier", func(r *activerecord.R) {
//		r.Validates("password", &activerecord.Exclusion{From: activesupport.Strings("12345678")})
//	})
type Exclusion struct {
	From activesupport.Slice

	AllowNil   bool
	AllowBlank bool
}

// AllowsNil returns true when nil values are allowed, and false otherwise.
func (e *Exclusion) AllowsNil() bool { return e.AllowNil }

// AllowsBlank returns true when blank values are allowed, and false otherwise.
func (e *Exclusion) AllowsBlank() bool { return e.AllowBlank }

// ValidateAttribute validates that the specified value is not in a slice. When it is,
// method returns ErrInvalidValue error.
func (e *Exclusion) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	if e.From.Contains(val) {
		return ErrInvalidValue{AttrName: attrName, Value: val, Message: "is reserved"}
	}
	return nil
}

// Length validates that the specified value of the attribute match the length
// restrictions supplied.
//
//	Supplier := activerecord.New("supplier", func(r *activerecord.R) {
//		r.Validates("phone", &activerecord.Length{Minumum: 7, Maximum: 32})
//		r.Validates("zip_code", &activerecord.Length{Minimum: 5})
//	})
type Length struct {
	// Minimum is the minimum length of the attribute.
	Minimum int
	// Maximum is the maximum length of the attribute.
	Maximum int

	// AllowNil skips validation, when attribute is nil.
	AllowNil bool
	// AllowBlank skips validation, when attribute is blank.
	AllowBlank bool
}

// AllowsNil returns true when nil values are allowed, and false otherwise.
func (l *Length) AllowsNil() bool { return l.AllowNil }

// AllowsBlank returns true when blank values are allowed, and false otherwise.
func (l *Length) AllowsBlank() bool { return l.AllowBlank }

// Initialize ensures the correctness of the parameters. When parameters are not valid:
// both `Mininum` and `Maximum` must be positive numbers, and `Minimum < Maximum`,
// method returns ErrArgument error.
func (l *Length) Initialize() error {
	if l.Maximum < l.Minimum {
		return activesupport.ErrArgument{
			Message: "length: maximum can't be less than minimum",
		}
	}
	if l.Minimum < 0 {
		return activesupport.ErrArgument{
			Message: "length: minimum can't be less than 0",
		}
	}
	return nil
}

// ValidateAttribute validates that the specified value comply the length restrictions.
//
// Method returns ErrInvalidType for non-character-based types and ErrInvalidValue, when
// length restrictions are not met.
func (l *Length) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	var length int
	switch val := val.(type) {
	case string:
		length = len(val)
	case []byte:
		length = len(val)
	case []rune:
		length = len(val)
	default:
		return ErrInvalidType{AttrName: attrName, TypeName: "String", Value: val}
	}
	if length < l.Minimum {
		return ErrInvalidValue{
			AttrName: attrName,
			Value:    val,
			Message:  fmt.Sprintf("is too short (minimum is %d characters)", l.Minimum),
		}
	}
	if length > l.Maximum {
		return ErrInvalidValue{
			AttrName: attrName,
			Value:    val,
			Message:  fmt.Sprintf("is too long (maximum is %d characters)", l.Maximum),
		}
	}
	return nil
}
