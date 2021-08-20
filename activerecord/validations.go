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

func (m validatorsMap) include(attrName string, validator AttributeValidator) {
	validators := m[attrName]
	m[attrName] = append(validators, validator)
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
		value := rec.AccessAttribute(attrName)

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

type IntValidator func(v int64) error

func (v IntValidator) AllowsNil() bool   { return true }
func (v IntValidator) AllowsBlank() bool { return true }

func (v IntValidator) ValidateAttribute(rec *ActiveRecord, attrName string, val interface{}) error {
	var intval int64
	switch val := val.(type) {
	case int:
		intval = int64(val)
	case int32:
		intval = int64(val)
	case int64:
		intval = val
	default:
		return ErrInvalidType{AttrName: attrName, TypeName: Int, Value: val}
	}
	if v != nil {
		return v(intval)
	}
	return nil
}

type StringValidator func(s string) error

func (v StringValidator) AllowsNil() bool   { return true }
func (v StringValidator) AllowsBlank() bool { return true }

func (v StringValidator) ValidateAttribute(rec *ActiveRecord, attrName string, val interface{}) error {
	s, ok := val.(string)
	if !ok {
		return ErrInvalidType{AttrName: attrName, TypeName: String, Value: val}
	}
	if v != nil {
		return v(s)
	}
	return nil
}

type FloatValidator func(f float64) error

func (v FloatValidator) AllowsNil() bool   { return true }
func (v FloatValidator) AllowsBlank() bool { return true }

func (v FloatValidator) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	f, ok := val.(float64)
	if !ok {
		return ErrInvalidType{AttrName: attrName, TypeName: Float, Value: val}
	}
	if v != nil {
		return v(f)
	}
	return nil
}

type BooleanValidator func(b bool) error

func (v BooleanValidator) AllowsNil() bool   { return true }
func (v BooleanValidator) AllowsBlank() bool { return true }

func (v BooleanValidator) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	b, ok := val.(bool)
	if !ok {
		return ErrInvalidType{AttrName: attrName, TypeName: Boolean, Value: val}
	}
	if v != nil {
		return v(b)
	}
	return nil
}

// Presence validate that specified value is not blank.
type Presence struct {
	AllowNil   bool
	AllowBlank bool
}

func (p *Presence) AllowsNil() bool   { return p.AllowNil }
func (p *Presence) AllowsBlank() bool { return p.AllowBlank }

func (p *Presence) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	if activesupport.IsBlank(val) {
		return ErrInvalidValue{AttrName: attrName, Value: val, Message: "can't be blank"}
	}
	return nil
}

type Format struct {
	With    activesupport.String
	Without activesupport.String

	AllowNil   bool
	AllowBlank bool

	re *regexp.Regexp
}

func (f *Format) AllowsNil() bool   { return f.AllowNil }
func (f *Format) AllowsBlank() bool { return f.AllowBlank }

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

func (f *Format) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	s, ok := val.(string)
	if !ok {
		return ErrInvalidType{AttrName: attrName, TypeName: String, Value: val}
	}

	match := f.re.Match([]byte(s))
	if (match && !f.Without.IsEmpty()) || (!match && f.Without.IsEmpty()) {
		return ErrInvalidValue{
			AttrName: attrName, Value: val, Message: "has invalid format",
		}
	}
	return nil
}

type Inclusion struct {
	In activesupport.Slice

	AllowNil   bool
	AllowBlank bool
}

func (i *Inclusion) AllowsNil() bool   { return i.AllowNil }
func (i *Inclusion) AllowsBlank() bool { return i.AllowBlank }

func (i *Inclusion) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	if !i.In.Contains(val) {
		return ErrInvalidValue{
			AttrName: attrName, Value: val, Message: "is not included in the list",
		}
	}
	return nil
}

type Exclusion struct {
	From activesupport.Slice

	AllowNil   bool
	AllowBlank bool
}

func (e *Exclusion) AllowsNil() bool   { return e.AllowNil }
func (e *Exclusion) AllowsBlank() bool { return e.AllowBlank }

func (e *Exclusion) ValidateAttribute(r *ActiveRecord, attrName string, val interface{}) error {
	if e.From.Contains(val) {
		return ErrInvalidValue{AttrName: attrName, Value: val, Message: "is reserved"}
	}
	return nil
}

type Length struct {
	Minimum int
	Maximum int

	// AllowNil skips validation, when attribute is nil.
	AllowNil   bool
	AllowBlank bool
}

func (l *Length) AllowsNil() bool   { return l.AllowNil }
func (l *Length) AllowsBlank() bool { return l.AllowBlank }

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
		return ErrInvalidType{AttrName: attrName, TypeName: String, Value: val}
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
