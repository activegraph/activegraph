package graphql

import (
	"encoding/json"

	graphql "github.com/vektah/gqlparser/v2/ast"

	"github.com/activegraph/activegraph/activesupport"
)

type SchemaReflect struct {
	schema *graphql.Schema
}

type TypeReflect struct {
	value  *graphql.Type
	schema *graphql.Schema
	depth  int
}

func MakeType(schema *graphql.Schema, t *graphql.Type) *TypeReflect {
	return &TypeReflect{value: t, schema: schema, depth: 7}
}

func (t *TypeReflect) Kind() string {
	if t.value.NonNull {
		return "NON_NULL"
	}
	if t.value.Elem != nil {
		return "LIST"
	}
	return string(t.schema.Types[t.value.NamedType].Kind)
}

func (t *TypeReflect) Introspect() activesupport.Hash {
	result := activesupport.Hash{
		"kind": t.Kind(), "name": t.value.Name(), "ofType": nil,
	}

	nextResult := result
	elem := t.value.Elem
	for i := 0; i < t.depth && elem != nil; i++ {
		t := MakeType(t.schema, elem)
		ofType := activesupport.Hash{
			"kind": t.Kind(), "name": t.value.Name(), "ofType": nil,
		}

		nextResult["ofType"] = ofType
		nextResult = ofType
		elem = elem.Elem
	}

	return result
}

func introspectFields(fields graphql.FieldList, schema *graphql.Schema) []activesupport.Hash {
	fieldsIntrospection := make([]activesupport.Hash, 0, len(fields))

	for _, def := range fields {
		fieldsIntrospection = append(fieldsIntrospection, activesupport.Hash{
			"name":             def.Name,
			"description":      def.Description,
			"args":             make([]activesupport.Hash, 0),
			"type":             MakeType(schema, def.Type).Introspect(),
			"isDeprecated":     false,
			"deprecatedReason": nil,
		})
	}

	return fieldsIntrospection
}

func introspect(schema *graphql.Schema) activesupport.Hash {
	typesIntrospection := make([]activesupport.Hash, 0, len(schema.Types))
	for _, def := range schema.Types {
		typesIntrospection = append(typesIntrospection, activesupport.Hash{
			"kind":          def.Kind,
			"name":          def.Name,
			"description":   def.Description,
			"fields":        introspectFields(def.Fields, schema),
			"inputFields":   nil,
			"interfaces":    make([]activesupport.Hash, 0),
			"enumValues":    nil,
			"possibleTypes": nil,
		})
	}

	schemaIntrospection := activesupport.Hash{
		"queryType": activesupport.Hash{"name": "Query"},
		"types":     typesIntrospection,
	}

	return activesupport.Hash{
		"data": activesupport.Hash{
			"__schema": schemaIntrospection,
		},
	}
}

func IntrospectionHandler(rw ResponseWriter, r *Request) {
	query := r.query.Operations.ForName("IntrospectionQuery")
	if query == nil {
		return
	}

	// schema := introspection.WrapSchema(r.schema)
	hash := introspect(r.schema)

	b, err := json.Marshal(hash)
	if err != nil {
		panic(err)
	}
	rw.Write(b)
}
