package activerecord

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/activegraph/activegraph/activesupport"
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

type ErrUnsupportedType struct {
	TypeName string
}

func (e ErrUnsupportedType) Error() string {
	return fmt.Sprintf("unsupported type '%s'", e.TypeName)
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

func (*Int64) String() string { return "int64" }

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

func (*String) String() string { return "string" }

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

func (*Float64) String() string { return "float64" }
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

func (b *Boolean) String() string { return "boolean" }

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

const (
	iso8601     = time.RFC3339Nano
	iso8601Date = "2006-01-02"
	iso8601Time = "15:04.05.999999999Z07:00"
)

func parseTime(layout string, value interface{}) (interface{}, error) {
	var (
		parsedTime time.Time
		err        error
	)
	switch value := value.(type) {
	case string:
		parsedTime, err = time.Parse(layout, value)
	case time.Time:
		parsedTime, err = value.UTC(), nil
	default:
		err = ErrType{Value: value}
	}
	if err != nil {
		return nil, err
	}
	return parsedTime.UTC(), nil
}

func formatTime(layout string, value interface{}) (interface{}, error) {
	switch value := value.(type) {
	case time.Time:
		return value.Format(layout), nil
	default:
		return nil, ErrType{Value: value}
	}
}

type DateTime struct{}

func (*DateTime) String() string { return "datetime" }

func (dt *DateTime) Deserialize(value interface{}) (interface{}, error) {
	value, err := parseTime(iso8601, value)
	if err != nil {
		return nil, ErrType{TypeName: dt.String(), Value: value}
	}
	return value, nil
}

func (dt *DateTime) Serialize(value interface{}) (interface{}, error) {
	value, err := formatTime(iso8601, value)
	if err != nil {
		return nil, ErrType{TypeName: dt.String(), Value: value}
	}
	return value, nil
}

type Date struct{}

func (*Date) String() string { return "date" }

func (d *Date) Deserialize(value interface{}) (interface{}, error) {
	value, err := parseTime(iso8601Date, value)
	if err != nil {
		return nil, ErrType{TypeName: d.String(), Value: value}
	}
	return value, nil
}

func (d *Date) Serialize(value interface{}) (interface{}, error) {
	value, err := formatTime(iso8601Date, value)
	if err != nil {
		return nil, ErrType{TypeName: d.String(), Value: value}
	}
	return value, nil
}

type Time struct{}

func (*Time) String() string { return "time" }

func (t *Time) Deserialize(value interface{}) (interface{}, error) {
	value, err := parseTime(iso8601Time, value)
	if err != nil {
		return nil, ErrType{TypeName: t.String(), Value: value}
	}
	return value, nil
}

func (t *Time) Serialize(value interface{}) (interface{}, error) {
	value, err := formatTime(iso8601Time, value)
	if err != nil {
		return nil, ErrType{TypeName: t.String(), Value: value}
	}
	return value, nil
}

type JSON struct{}

func (*JSON) String() string { return "json" }

func (j *JSON) Deserialize(value interface{}) (interface{}, error) {
	var (
		hash activesupport.Hash
		err  error
	)
	switch value := value.(type) {
	case string:
		err = json.Unmarshal([]byte(value), &hash)
	case []byte:
		err = json.Unmarshal(value, &hash)
	default:
		return nil, ErrType{TypeName: j.String(), Value: value}
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (j *JSON) Serialize(value interface{}) (interface{}, error) {
	return json.Marshal(value)
}
