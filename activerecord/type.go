package activerecord

import (
	"fmt"
)

type Type interface {
	Deserialize(value interface{}) (interface{}, error)
	Serialize(value interface{}) (interface{}, error)
}

type ErrType struct {
	TypeName string
	Value    interface{}
}

func (e ErrType) Error() string {
	return fmt.Sprintf("invalid value '%v' for %s type", e.Value, e.TypeName)
}

type null struct {
	Type
}

func (n null) Deserialize(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	return n.Type.Deserialize(value)
}

type Int64 struct{}

func (*Int64) Deserialize(value interface{}) (interface{}, error) {
	var intval int64
	switch value := value.(type) {
	case int:
		intval = int64(value)
	case int32:
		intval = int64(value)
	case int64:
		intval = value
	default:
		return nil, ErrType{TypeName: "Integer", Value: value}
	}
	return intval, nil
}

func (*Int64) Serialize(value interface{}) (interface{}, error) {
	return value, nil
}

type String struct{}

func (*String) Deserialize(value interface{}) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, ErrType{TypeName: "String", Value: value}
	}
	return s, nil
}

func (*String) Serialize(value interface{}) (interface{}, error) {
	return value, nil
}

type Float64 struct{}

func (*Float64) Deserialize(value interface{}) (interface{}, error) {
	f, ok := value.(float64)
	if !ok {
		return nil, ErrType{TypeName: "Float64", Value: value}
	}
	return f, nil
}

func (*Float64) Serialize(value interface{}) (interface{}, error) {
	return value, nil
}

type Boolean struct{}

func (*Boolean) Deserialize(value interface{}) (interface{}, error) {
	b, ok := value.(bool)
	if !ok {
		return nil, ErrType{TypeName: "Boolean", Value: value}
	}
	return b, nil
}

func (*Boolean) Serialize(value interface{}) (interface{}, error) {
	return value, nil
}
