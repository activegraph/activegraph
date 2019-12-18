package relsy

import (
	"encoding/json"
	"reflect"
	"strings"
)

func jsonName(sf reflect.StructField) (name string, ignore bool) {
	tag, ok := sf.Tag.Lookup("json")
	values := strings.Split(tag, ",")

	// When the JSON tag is not defined, or name is not defined explicitely,
	// use field name without any modifications at all.
	if !ok || len(values) == 0 || values[0] == "" {
		return sf.Name, false
	}

	if values[0] == "-" && len(values) == 1 {
		return "", true
	}
	return values[0], false
}

func jsonUnpack(m map[string]interface{}, dest interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}
