package activerecord

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/activegraph/activegraph/activesupport"
)

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

type validations struct {
	validators validatorsMap
	errs       map[string][]error
}

func newValidations(validators validatorsMap) *validations {
	return &validations{
		validators: validators.copy(),
		errs:       make(map[string][]error),
	}
}

func (v *validations) copy() *validations {
	return newValidations(v.validators)
}

func (v *validations) validate(rec *ActiveRecord) error {
	for attr, validators := range v.validators {
		for _, validator := range validators {
			err := validator.Validate(rec, attr, rec.AccessAttribute(attr))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *validations) Errors(attrName ...string) []error {
	return nil
}

func (v *validations) ClearErrors() {
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
			AttrName: attrName, Value: val, Message: "is not present (blank or nil)",
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
	if match && v.options.Without {
		return ErrInvalidValue{
			AttrName: attrName, Value: val, Message: "match regexp",
		}
	}
	if !match && !v.options.Without {
		return ErrInvalidValue{
			AttrName: attrName, Value: val, Message: "do not match regexp",
		}
	}
	return nil
}
