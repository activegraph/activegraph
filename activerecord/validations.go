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
	return fmt.Sprintf("Model %v invalid%s", e.Model, errors)
}

type ErrInvalidValue struct {
	AttrName string
	Message  string
	Value    interface{}
}

func (e ErrInvalidValue) Error() string {
	text := fmt.Sprintf("invalid value '%v' for attribute '%s'", e.Value, e.AttrName)
	if len(e.Message) != 0 {
		return fmt.Sprintf("%s, %s", text, e.Message)
	}
	return text
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
	if e.errors != nil {
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

func MaxLen(num int) StringValidator {
	if num < 0 {
		panic("num is less zero")
	}
	return func(s string) error {
		if len(s) > num {
			return errors.Errorf("%q lenght is >%d", s, num)
		}
		return nil
	}
}

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

// PresenceValidator validate that specified value is not blank.
type PresenceValidator struct{}

func (v PresenceValidator) Validate(r *ActiveRecord, attrName string, val interface{}) error {
	var blank bool

	switch val := val.(type) {
	case string:
		blank = activesupport.String(val).IsBlank()
	case []rune:
		blank = activesupport.String(string(val)).IsBlank()
	case []byte:
		blank = activesupport.String(string(val)).IsBlank()
	case nil:
		blank = true
	}

	if blank {
		return ErrInvalidValue{
			AttrName: attrName, Value: val, Message: "can't be blank",
		}
	}
	return nil
}

type FormatOptions struct {
	Without bool
}

type FormatValidator struct {
	re      *regexp.Regexp
	options FormatOptions
}

func NewFormatValidator(re string, options ...FormatOptions) (*FormatValidator, error) {
	var fv FormatValidator
	switch len(options) {
	case 0:
	case 1:
		fv.options = options[0]
	default:
		return nil, &activesupport.ErrMultipleVariadicArguments{Name: "options"}
	}

	reCompiled, err := regexp.Compile(re)
	if err != nil {
		return nil, err
	}
	fv.re = reCompiled
	return &fv, nil
}

func (v FormatValidator) Validate(r *ActiveRecord, attrName string, val interface{}) error {
	s, ok := val.(string)
	if !ok {
		return ErrInvalidType{AttrName: attrName, TypeName: String, Value: val}
	}

	match := v.re.Match([]byte(s))
	if (match && v.options.Without) || (!match && !v.options.Without) {
		return ErrInvalidValue{
			AttrName: attrName, Value: val, Message: "is invalid",
		}
	}
	return nil
}

type InclusionValidator struct {
	in activesupport.Slice
}

func NewInclusionValidator(in activesupport.Slice) InclusionValidator {
	return InclusionValidator{in: in}
}

func (v InclusionValidator) Validate(r *ActiveRecord, attrName string, val interface{}) error {
	if !v.in.Contains(val) {
		return ErrInvalidValue{
			AttrName: attrName, Value: val, Message: "is not included in the list",
		}
	}
	return nil
}

type ExclusionValidator struct {
	from activesupport.Slice
}

func NewExclusionValidator(from activesupport.Slice) ExclusionValidator {
	return ExclusionValidator{from: from}
}

func (v ExclusionValidator) Validate(r *ActiveRecord, attrName string, val interface{}) error {
	if v.from.Contains(val) {
		return ErrInvalidValue{
			AttrName: attrName, Value: val, Message: "is reserved",
		}
	}
	return nil
}
