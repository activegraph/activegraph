package relsy

import (
	"errors"
	"reflect"

	"github.com/graphql-go/graphql"
)

// objectType defines the "direction" of the object, it's either input
// or output object.
type objectType int

const (
	inObjectType objectType = iota
	outObjectType
)

// GraphQLCompiler is GraphQL schema compiler, it produces GraphQL
// schema definition from the Go type and function definitions.
type GraphQLCompiler struct {
	inputs    map[string]graphql.Type
	outputs   map[string]graphql.Type
	queries   graphql.Fields
	mutations graphql.Fields
}

func (c *GraphQLCompiler) init() {
	if c.inputs == nil {
		c.inputs = make(map[string]graphql.Type)
	}
	if c.outputs == nil {
		c.outputs = make(map[string]graphql.Type)
	}
	if c.queries == nil {
		c.queries = make(graphql.Fields)
	}
	if c.mutations == nil {
		c.mutations = make(graphql.Fields)
	}
}

// AddType registers the given type in the GraphQL schema.
func (c *GraphQLCompiler) AddType(typedef TypeDef) error {
	c.init()

	if _, exist := c.outputs[typedef.Name]; exist {
		return errors.New("relsy: multiple type registrations for " + typedef.Name)
	}

	// Create a new GraphQL object from the Go type definition.
	gqltype, err := newGraphQLType(typedef.Type, outObjectType, c.outputs)
	if err != nil {
		return err
	}

	obj, isObject := gqltype.(*graphql.Object)
	if !isObject {
		return errors.New("relsy: type expected to be an object")
	}

	// Add methods for a new GraphQL type. All methods should be
	// bounded to this GraphQL type.
	for name, funcdef := range typedef.Funcs {
		out, err := newGraphQLType(funcdef.Out, outObjectType, c.outputs)
		if err != nil {
			return err
		}

		obj.AddFieldConfig(name, &graphql.Field{
			Name:    name,
			Type:    out,
			Resolve: newGraphQLBoundFunc(funcdef),
		})
	}

	c.outputs[typedef.Name] = obj
	return nil
}

func (c *GraphQLCompiler) addFunc(funcdef FuncDef, registry graphql.Fields) error {
	if _, exist := registry[funcdef.Name]; exist {
		return errors.New("relsy: multiple registrations for " + funcdef.Name)
	}

	in, err := newGraphQLArguments(funcdef.In, c.inputs)
	if err != nil {
		return err
	}
	out, err := newGraphQLType(funcdef.Out, outObjectType, c.outputs)
	if err != nil {
		return err
	}

	registry[funcdef.Name] = &graphql.Field{
		Args:    in,
		Type:    out,
		Resolve: newGraphQLUnboundFunc(funcdef),
	}
	return nil
}

func (c *GraphQLCompiler) AddQuery(funcdef FuncDef) error {
	c.init()
	return c.addFunc(funcdef, c.queries)
}

func (c *GraphQLCompiler) AddMutation(funcdef FuncDef) (err error) {
	c.init()
	return c.addFunc(funcdef, c.mutations)
}

// Compile creates GraphQL schema based on registered types, queries and
// mutations.
func (c *GraphQLCompiler) Compile() (graphql.Schema, error) {
	c.init()

	var (
		query    *graphql.Object
		mutation *graphql.Object
	)

	if len(c.queries) != 0 {
		query = graphql.NewObject(graphql.ObjectConfig{
			Name: "Query", Fields: c.queries,
		})
	}
	if len(c.mutations) != 0 {
		mutation = graphql.NewObject(graphql.ObjectConfig{
			Name: "Mutation", Fields: c.mutations,
		})
	}

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    query,
		Mutation: mutation,
	})
}

// newGraphQLBoundFunc creates a field resolve function that can be
// used as a method of the type.
func newGraphQLBoundFunc(funcdef FuncDef) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		return funcdef.CallBound(p.Context, p.Source)
	}
}

// newGraphQLUnboundFunc creates a field resolve function that can be
// used as GraphQL query.
func newGraphQLUnboundFunc(funcdef FuncDef) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		return funcdef.Call(p.Context, p.Args)
	}
}

func newGraphQLArguments(
	gotype reflect.Type, types map[string]graphql.Type,
) (
	graphql.FieldConfigArgument, error,
) {
	if gotype == nil {
		return nil, nil
	}
	gqltype, err := newGraphQLType(gotype, inObjectType, types)
	if err != nil {
		return nil, err
	}
	obj, ok := gqltype.(*graphql.InputObject)
	if !ok {
		return nil, errors.New("argument type is expected to be an object")
	}

	var (
		fields = obj.Fields()
		args   = make(graphql.FieldConfigArgument, len(fields))
	)
	for name, field := range fields {
		args[name] = &graphql.ArgumentConfig{Type: field.Type}
	}
	return args, nil
}

// newGraphQLObject returns a new GraphQL object with the given name and
// the set of fields. Object type specifies the type: either input or
// output object.
func newGraphQLObject(
	ot objectType, name string, fields map[string]graphql.Type,
) graphql.Type {
	switch ot {
	case inObjectType:
		objFields := make(graphql.InputObjectConfigFieldMap)
		for name, t := range fields {
			objFields[name] = &graphql.InputObjectFieldConfig{Type: t}
		}
		return graphql.NewInputObject(graphql.InputObjectConfig{
			Name:   name,
			Fields: objFields,
		})
	case outObjectType:
		objFields := make(graphql.Fields)
		for name, t := range fields {
			objFields[name] = &graphql.Field{Name: name, Type: t}
		}
		return graphql.NewObject(graphql.ObjectConfig{
			Name:   name,
			Fields: objFields,
		})
	default:
		return nil
	}
}

// newGraphQLType creates a new GraphQL type from the Go type, it recursively
// traverses complex types, like slices, arrays and structures.
func newGraphQLType(
	gotype reflect.Type, ot objectType, types map[string]graphql.Type,
) (
	gqltype graphql.Type, err error,
) {
	switch gotype.Kind() {
	case reflect.Float32, reflect.Float64:
		return graphql.Float, nil
	case reflect.Int32, reflect.Int64:
		return graphql.Int, nil
	case reflect.String:
		return graphql.String, nil
	case reflect.Slice, reflect.Array:
		subtype, err := newGraphQLType(gotype.Elem(), ot, types)
		if err != nil {
			return gqltype, err
		}
		return graphql.NewList(subtype), nil
	case reflect.Struct:
		// When the passed object is a structure, look it up in the passed
		// list of registered types and choose it in order to prevent
		// multiple declarations of the same type.
		if gqltype, ok := types[gotype.Name()]; ok {
			return gqltype, nil
		}

		fields := make(map[string]graphql.Type)
		for i := 0; i < gotype.NumField(); i++ {
			field := gotype.Field(i)
			fieldName, skip := jsonName(field)

			// Skip the field, when it is ignored from the JSON representation.
			if skip {
				continue
			}
			subtype, err := newGraphQLType(field.Type, ot, types)
			if err != nil {
				return gqltype, err
			}
			fields[fieldName] = subtype
		}

		// Ensure that registry of types is updated with a new type.
		obj := newGraphQLObject(ot, gotype.Name(), fields)
		types[obj.Name()] = obj
		return obj, nil
	default:
		return gqltype, errors.New("relsy: unsupported type " + gotype.Name())
	}
}
