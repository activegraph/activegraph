package activerecord

import (
	"fmt"

	"github.com/pkg/errors"
)

type ErrInvalidValue struct {
	TypeName string
	Value    interface{}
}

func (e ErrInvalidValue) Error() string {
	return fmt.Sprintf("invalid value '%v' for %s type", e.Value, e.TypeName)
}

type Validator interface {
	Validate(v interface{}) error
}

type IntValidator func(v int) error

type IntValidators []IntValidator

func ValidatesInt(vv ...IntValidator) IntValidators { return vv }

func (vv IntValidators) Validate(v interface{}) error {
	if v == nil {
		return nil
	}

	var intval int
	switch v := v.(type) {
	case int:
		intval = v
	case int32:
		intval = int(v)
	case int64:
		intval = int(v)
	default:
		return ErrInvalidValue{TypeName: Int, Value: v}
	}

	for i := 0; i < len(vv); i++ {
		if err := vv[i](intval); err != nil {
			return err
		}
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

type StringValidators []StringValidator

func ValidatesString(vv ...StringValidator) StringValidators { return vv }

func (vv StringValidators) Validate(v interface{}) error {
	if v == nil {
		return nil
	}

	val, ok := v.(string)
	if !ok {
		return ErrInvalidValue{TypeName: String, Value: v}
	}

	for i := 0; i < len(vv); i++ {
		if err := vv[i](val); err != nil {
			return err
		}
	}
	return nil
}

type FloatValidator func(f float64) error

type FloatValidators []FloatValidator

func ValidatesFloat(vv ...FloatValidator) FloatValidators { return vv }

func (vv FloatValidators) Validate(v interface{}) error {
	if v == nil {
		return nil
	}

	val, ok := v.(float64)
	if !ok {
		return ErrInvalidValue{TypeName: Float, Value: v}
	}

	for i := 0; i < len(vv); i++ {
		if err := vv[i](val); err != nil {
			return err
		}
	}
	return nil
}

type BooleanValidator func(b bool) error

type BooleanValidators []BooleanValidator

func ValidatesBoolean(vv ...BooleanValidator) BooleanValidators { return vv }

func (vv BooleanValidators) Validate(v interface{}) error {
	if v == nil {
		return nil
	}

	val, ok := v.(bool)
	if !ok {
		return ErrInvalidValue{TypeName: Boolean, Value: v}
	}

	for i := 0; i < len(vv); i++ {
		if err := vv[i](val); err != nil {
			return err
		}
	}
	return nil
}
