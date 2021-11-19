package graphql

import (
	"fmt"
	"strconv"
	"time"

	graphql "github.com/vektah/gqlparser/v2/ast"

	"github.com/activegraph/activegraph/activesupport"
)

var DefaultDecoder = new(Decoder)

func init() {
	DefaultDecoder.RegisterName(Int.Name, UnmarshalerFunc(UnmarshalInt))
	DefaultDecoder.RegisterName(Float.Name, UnmarshalerFunc(UnmarshalFloat))
	DefaultDecoder.RegisterName(String.Name, UnmarshalerFunc(UnmarshalString))
	DefaultDecoder.RegisterName(Boolean.Name, UnmarshalerFunc(UnmarshalBoolean))
	// Non-standard types.
	DefaultDecoder.RegisterName(DateTime.Name, UnmarshalerFunc(UnmarshalDateTime))
}

// ErrUnsupportedType is returned by Decoder when attempting to decode
// an unsupported value type.
type ErrUnsupportedType struct {
	*graphql.Type
}

func (e ErrUnsupportedType) Error() string {
	return "graphql: unsupported type " + e.Type.String()
}

type Unmarshaler interface {
	Unmarshal(raw string) (interface{}, error)
}

// UnmarshalerFunc is a function adapter for Unmarshaler interface.
type UnmarshalerFunc func(raw string) (interface{}, error)

func (fn UnmarshalerFunc) Unmarshal(raw string) (interface{}, error) {
	return fn(raw)
}

// Decoder decodes GraphQL values.
type Decoder struct {
	types map[string]Unmarshaler
}

// RegisterName registers the type unmarshaler under the given name.
func (d *Decoder) RegisterName(name string, unmarshaler Unmarshaler) {
	if name == "" {
		panic("graphql: attempt to register empty name")
	}
	if _, dup := d.types[name]; dup {
		panic(fmt.Sprintf("graphql: registering duplicate name %q", name))
	}
	if d.types == nil {
		d.types = make(map[string]Unmarshaler)
	}
	d.types[name] = unmarshaler
}

func (d *Decoder) Decode(v *graphql.Value) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	if v.ExpectedType == nil {
		panic("graphql: attempt to decode unvalidated value")
	}

	switch v.Kind {
	case graphql.NullValue:
		return nil, nil
	case graphql.ListValue:
		list := make([]interface{}, len(v.Children))
		for i, child := range v.Children {
			element, err := d.Decode(child.Value)
			if err != nil {
				return list, err
			}
			list[i] = element
		}
		return list, nil
	case graphql.ObjectValue:
		object := activesupport.Hash{}
		for _, child := range v.Children {
			element, err := d.Decode(child.Value)
			if err != nil {
				return object, err
			}
			object[child.Name] = element
		}
		return object, nil
	}

	decoder, ok := d.types[v.ExpectedType.Name()]
	if !ok {
		return nil, ErrUnsupportedType{v.ExpectedType}
	}
	return decoder.Unmarshal(v.Raw)
}

// Int is a scalar type that represents a signed 32-bit numeric non-fractional value.
var Int = &graphql.Definition{
	Kind:        graphql.Scalar,
	Name:        "Int",
	Description: "A signed 32-bit numeric non-fractional value.",
}

func UnmarshalInt(raw string) (interface{}, error) {
	return strconv.ParseInt(raw, 10, 32)
}

// Float is a scalar type that represents signed double-precision finite values as
// specified by IEEE 754.
var Float = &graphql.Definition{
	Kind:        graphql.Scalar,
	Name:        "Float",
	Description: "A signed double-precision finite value as specified by IEEE 754.",
}

func UnmarshalFloat(raw string) (interface{}, error) {
	return strconv.ParseFloat(raw, 64)
}

// String is a scalar type that represents a sequence of Unicode code points.
var String = &graphql.Definition{
	Kind:        graphql.Scalar,
	Name:        "String",
	Description: "A sequence of Unicode code points.",
}

func UnmarshalString(raw string) (interface{}, error) {
	return raw, nil
}

// Boolean is a scalar type that represents true or false.
var Boolean = &graphql.Definition{
	Kind:        graphql.Scalar,
	Name:        "Boolean",
	Description: "Represents true of false.",
}

func UnmarshalBoolean(raw string) (interface{}, error) {
	return strconv.ParseBool(raw)
}

const (
	iso8601     = time.RFC3339Nano
	iso8601Date = "2006-01-02"
	iso8601Time = "15:04.05.999999999Z07:00"
)

var DateTime = &graphql.Definition{
	Kind: graphql.Scalar,
	Name: "DateTime",
}

func UnmarshalDateTime(raw string) (interface{}, error) {
	return time.Parse(iso8601, raw)
}
