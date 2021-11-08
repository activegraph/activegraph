package graphql

import (
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

func introspectArgs(args graphql.ArgumentDefinitionList, schema *graphql.Schema) []activesupport.Hash {
	argsIntrospection := make([]activesupport.Hash, 0, len(args))

	for _, def := range args {
		argsIntrospection = append(argsIntrospection, activesupport.Hash{
			"name":        def.Name,
			"description": def.Description,
			"type":        MakeType(schema, def.Type).Introspect(),
			// TODO: add support of default value.
			"defaultValue": nil,
		})
	}

	return argsIntrospection
}

func introspectFields(fields graphql.FieldList, schema *graphql.Schema) []activesupport.Hash {
	fieldsIntrospection := make([]activesupport.Hash, 0, len(fields))

	for _, def := range fields {
		fieldsIntrospection = append(fieldsIntrospection, activesupport.Hash{
			"name":             def.Name,
			"description":      def.Description,
			"args":             introspectArgs(def.Arguments, schema),
			"type":             MakeType(schema, def.Type).Introspect(),
			"isDeprecated":     false,
			"deprecatedReason": nil,
		})
	}

	return fieldsIntrospection
}

func introspectInputFields(fields graphql.FieldList, schema *graphql.Schema) []activesupport.Hash {
	fieldsIntrospection := make([]activesupport.Hash, 0, len(fields))
	for _, def := range fields {
		fieldsIntrospection = append(fieldsIntrospection, activesupport.Hash{
			"name":         def.Name,
			"description":  def.Description,
			"type":         MakeType(schema, def.Type).Introspect(),
			"defaultValue": nil,
		})
	}

	return fieldsIntrospection
}

func introspect(schema *graphql.Schema) activesupport.Hash {
	typesIntrospection := make([]activesupport.Hash, 0, len(schema.Types))
	for _, def := range schema.Types {
		typeIntrospection := activesupport.Hash{
			"kind":          def.Kind,
			"name":          def.Name,
			"description":   def.Description,
			"fields":        nil,
			"inputFields":   nil,
			"interfaces":    make([]activesupport.Hash, 0),
			"enumValues":    nil,
			"possibleTypes": nil,
		}

		if def.Kind == graphql.InputObject {
			typeIntrospection["inputFields"] = introspectInputFields(def.Fields, schema)
		} else {
			typeIntrospection["fields"] = introspectFields(def.Fields, schema)
		}

		typesIntrospection = append(typesIntrospection, typeIntrospection)
	}

	schemaIntrospection := activesupport.Hash{
		"queryType":    activesupport.Hash{"name": "Query"},
		"mutationType": activesupport.Hash{"name": "Mutation"},
		"types":        typesIntrospection,
	}

	return schemaIntrospection
}

func IntrospectionHandler(rw ResponseWriter, r *Request) {
	query := r.query.Operations.ForName("IntrospectionQuery")
	if query == nil {
		return
	}
	rw.WriteData("__schema", introspect(r.schema))
}
