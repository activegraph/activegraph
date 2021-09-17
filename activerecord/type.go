package activerecord

import (
	"fmt"
)

type Type interface {
	fmt.Stringer

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

type Nil struct {
	Type
}

func (n Nil) String() string {
	return n.Type.String() + "?"
}

func (n Nil) Deserialize(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	return n.Type.Deserialize(value)
}

type Int64 struct{}

func (*Int64) String() string { return "Int64" }

func (i64 *Int64) Deserialize(value interface{}) (interface{}, error) {
	var intval int64
	switch value := value.(type) {
	case int:
		intval = int64(value)
	case int32:
		intval = int64(value)
	case int64:
		intval = value
	default:
		return nil, ErrType{TypeName: i64.String(), Value: value}
	}
	return intval, nil
}

func (*Int64) Serialize(value interface{}) (interface{}, error) {
	return value, nil
}

type String struct{}

func (*String) String() string { return "String" }

func (s *String) Deserialize(value interface{}) (interface{}, error) {
	strval, ok := value.(string)
	if !ok {
		return nil, ErrType{TypeName: s.String(), Value: value}
	}
	return strval, nil
}

func (*String) Serialize(value interface{}) (interface{}, error) {
	return value, nil
}

type Float64 struct{}

func (*Float64) String() string { return "Float64" }
func (f64 *Float64) Deserialize(value interface{}) (interface{}, error) {
	f, ok := value.(float64)
	if !ok {
		return nil, ErrType{TypeName: f64.String(), Value: value}
	}
	return f, nil
}

func (*Float64) Serialize(value interface{}) (interface{}, error) {
	return value, nil
}

type Boolean struct{}

func (b *Boolean) String() string { return "Boolean" }

func (b *Boolean) Deserialize(value interface{}) (interface{}, error) {
	boolval, ok := value.(bool)
	if !ok {
		return nil, ErrType{TypeName: b.String(), Value: value}
	}
	return boolval, nil
}

func (*Boolean) Serialize(value interface{}) (interface{}, error) {
	return value, nil
}
