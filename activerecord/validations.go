package activerecord

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

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
	Validate(r *ActiveRecord, attrName string, val interface{}) error
}

type ValidatorFunc func(r *ActiveRecord, attrName string, val interface{}) error

func (fn ValidatorFunc) Validate(r *ActiveRecord, attrName string, val interface{}) error {
	return fn(r, attrName, val)
}

type validate struct {
	first  *Presence
	second Validator
}

func (v *validate) Validate(r *ActiveRecord, attrName string, val interface{}) error {
	if v == nil {
		return errors.New("validation is not initialized, call 'Initialize'")
	}
	switch err := v.first.checkValidity(attrName, val).(type) {
	case activesupport.ErrNext:
	default:
		return err
	}
	return v.second.Validate(r, attrName, val)
}

type validatorsMap map[string][]Validator

func (m validatorsMap) copy() validatorsMap {
	mm := make(validatorsMap, len(m))
	for name, validators := range m {
		mm[name] = validators
	}
	return mm
}

func (m validatorsMap) include(attrName string, validator Validator) {
	validators := m[attrName]
	m[attrName] = append(validators, validator)
}

func (m validatorsMap) extend(attrNames []string, validator Validator) {
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

	for attr, validators := range v.validators {
		for _, validator := range validators {
			err := validator.Validate(rec, attr, rec.AccessAttribute(attr))
			if err != nil {
				v.errors.Add(attr, err)
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

func (v IntValidator) Validate(rec *ActiveRecord, attrName string, val interface{}) error {
	if val == nil {
		return nil
	}
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

func (v StringValidator) Validate(rec *ActiveRecord, attrName string, val interface{}) error {
	if val == nil {
		return nil
	}
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

func (v FloatValidator) Validate(r *ActiveRecord, attrName string, val interface{}) error {
	if val == nil {
		return nil
	}
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

func (v BooleanValidator) Validate(r *ActiveRecord, attrName string, val interface{}) error {
	if val == nil {
		return nil
	}
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
	AllowBlank bool
	allowNil   bool
}

func (p *Presence) checkValidity(attrName string, val interface{}) error {
	if val == nil {
		if !p.allowNil {
			return ErrInvalidValue{AttrName: attrName, Value: val, Message: "can't be nil"}
		} else {
			return nil
		}
	}

	var s activesupport.String

	switch val := val.(type) {
	case string:
		s = activesupport.String(val)
	case []rune:
		s = activesupport.String(string(val))
	case []byte:
		s = activesupport.String(string(val))
	default:
		return activesupport.ErrNext{}
	}

	if s.IsBlank() {
		if !p.AllowBlank {
			return &ErrInvalidValue{AttrName: attrName, Value: val, Message: "can't be blank"}
		} else {
			return nil
		}
	}
	return activesupport.ErrNext{}
}

func (p *Presence) Validate(r *ActiveRecord, attrName string, val interface{}) error {
	return p.checkValidity(attrName, val)
}

type Format struct {
	With    activesupport.String
	Without activesupport.String

	AllowNil   bool
	AllowBlank bool

	re *regexp.Regexp
	*validate
}

func (f *Format) Initialize() (err error) {
	f.validate = &validate{&Presence{f.AllowNil, f.AllowBlank}, ValidatorFunc(f.impl)}

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

func (f *Format) impl(r *ActiveRecord, attrName string, val interface{}) error {
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

	*validate
}

func (i *Inclusion) Initialize() error {
	i.validate = &validate{&Presence{i.AllowNil, i.AllowBlank}, ValidatorFunc(i.impl)}
	return nil
}

func (i *Inclusion) impl(r *ActiveRecord, attrName string, val interface{}) error {
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

	*validate
}

func (e *Exclusion) Initialize() error {
	e.validate = &validate{&Presence{e.AllowNil, e.AllowBlank}, ValidatorFunc(e.impl)}
	return nil
}

func (e *Exclusion) impl(r *ActiveRecord, attrName string, val interface{}) error {
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

	*validate
}

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
	l.validate = &validate{&Presence{l.AllowNil, l.AllowBlank}, ValidatorFunc(l.impl)}
	return nil
}

func (l *Length) impl(r *ActiveRecord, attrName string, val interface{}) error {
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
