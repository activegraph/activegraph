package resly

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

// GraphQL is GraphQL schema compiler, it produces GraphQL
// schema definition from the Go type and function definitions.
type GraphQL struct {
	inputs    map[string]graphql.Type
	outputs   map[string]graphql.Type
	queries   graphql.Fields
	mutations graphql.Fields
}

func (c *GraphQL) init() {
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
func (c *GraphQL) AddType(typedef TypeDef) error {
	c.init()

	if _, exist := c.outputs[typedef.Name]; exist {
		return errors.New("resly: multiple type registrations for " + typedef.Name)
	}

	// Create a new GraphQL object from the Go type definition.
	gqltype, err := newType(typedef.Type, outObjectType, c.outputs)
	if err != nil {
		return err
	}

	obj, isObject := graphql.GetNullable(gqltype).(*graphql.Object)
	if !isObject {
		return errors.New("resly: type expected to be an object")
	}

	// Add methods for a new GraphQL type. All methods should be
	// bounded to this GraphQL type.
	for name, funcdef := range typedef.Funcs {
		out, err := newType(funcdef.Out, outObjectType, c.outputs)
		if err != nil {
			return err
		}

		obj.AddFieldConfig(name, &graphql.Field{
			Name:    name,
			Type:    out,
			Resolve: newBoundFunc(funcdef),
		})
	}

	c.outputs[typedef.Name] = obj
	return nil
}

func (c *GraphQL) AddQuery(funcdef FuncDef) error {
	c.init()
	if _, dup := c.queries[funcdef.Name]; dup {
		return errors.New("resly: multiple registrations for " + funcdef.Name)
	}

	in, err := newQueryArgs(funcdef.In, c.inputs)
	if err != nil {
		return err
	}

	out, err := newType(funcdef.Out, outObjectType, c.outputs)
	if err != nil {
		return err
	}

	c.queries[funcdef.Name] = &graphql.Field{
		Args:    in,
		Type:    out,
		Resolve: newQueryFunc(funcdef),
	}
	return nil
}

func (c *GraphQL) AddMutation(funcdef FuncDef) (err error) {
	c.init()
	if _, dup := c.mutations[funcdef.Name]; dup {
		return errors.New("resly: multiple registrations for " + funcdef.Name)
	}

	in, err := newMutationArgs(funcdef.In, c.inputs)
	if err != nil {
		return err
	}

	out, err := newType(funcdef.Out, outObjectType, c.outputs)
	if err != nil {
		return err
	}

	c.mutations[funcdef.Name] = &graphql.Field{
		Args:    in,
		Type:    out,
		Resolve: newMutationFunc(funcdef),
	}
	return nil
}

// Compile creates GraphQL schema based on registered types, queries and
// mutations.
func (c *GraphQL) CreateSchema() (graphql.Schema, error) {
	c.init()

	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query", Fields: c.queries,
	})

	var mutation *graphql.Object
	if len(c.mutations) > 0 {
		mutation = graphql.NewObject(graphql.ObjectConfig{
			Name: "Mutation", Fields: c.mutations,
		})
	}

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    query,
		Mutation: mutation,
	})
}

// newBoundFunc creates a field resolve function that can be
// used as a method of the type.
func newBoundFunc(funcdef FuncDef) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		return funcdef.CallBound(p.Context, p.Source)
	}
}

// newMutationArgs creates configuration of the arguments for the mutation function.
//
// The specificity: all mutations must accept a single argument called "input", which
// type also must be an InputObject type.
func newMutationArgs(gotype reflect.Type, types map[string]graphql.Type) (
	graphql.FieldConfigArgument, error,
) {
	if gotype == nil {
		return nil, nil
	}
	gqltype, err := newType(gotype, inObjectType, types)
	if err != nil {
		return nil, err
	}

	return graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: gqltype},
	}, nil
}

// newMutationFunc creates a field resolve function that can be used as GraphQL mutation.
func newMutationFunc(funcdef FuncDef) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		// See the newMutationArgs for reference of the input parameters.
		input, ok := p.Args["input"]
		if !ok {
			return nil, errors.New("missing 'input' argument in the mutation " + funcdef.Name)
		}
		return funcdef.Call(p.Context, input)
	}
}

// newQueryArgs creates configuration of the arguments for the query function.
func newQueryArgs(gotype reflect.Type, types map[string]graphql.Type) (
	graphql.FieldConfigArgument, error,
) {
	if gotype == nil {
		return nil, nil
	}
	gqltype, err := newType(gotype, inObjectType, types)
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

func newQueryFunc(funcdef FuncDef) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		return funcdef.Call(p.Context, p.Args)
	}
}

// newObject returns a new GraphQL object with the given name and
// the set of fields. Object type specifies the type: either input or
// output object.
func newObject(
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

// newType creates a new GraphQL type from the Go type, it recursively
// traverses complex types, like slices, arrays and structures.
func newType(
	gotype reflect.Type, ot objectType, types map[string]graphql.Type,
) (
	gqltype graphql.Type, err error,
) {
	switch gotype.Kind() {
	case reflect.Ptr:
		gqltype, err = newType(gotype.Elem(), ot, types)
		if err != nil {
			return nil, err
		}
		return graphql.GetNullable(gqltype).(graphql.Type), nil
	case reflect.Float32, reflect.Float64:
		return graphql.NewNonNull(graphql.Float), nil
	case reflect.Uint, reflect.Uint32, reflect.Uint64, reflect.Int, reflect.Int32, reflect.Int64:
		return graphql.NewNonNull(graphql.Int), nil
	case reflect.String:
		return graphql.NewNonNull(graphql.String), nil
	case reflect.Slice, reflect.Array:
		subtype, err := newType(gotype.Elem(), ot, types)
		if err != nil {
			return gqltype, err
		}
		return graphql.NewNonNull(graphql.NewList(subtype)), nil
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
			subtype, err := newType(field.Type, ot, types)
			if err != nil {
				return gqltype, err
			}
			fields[fieldName] = subtype
		}

		// Ensure that registry of types is updated with a new type.
		obj := newObject(ot, gotype.Name(), fields)
		types[obj.Name()] = graphql.NewNonNull(obj)

		return types[obj.Name()], nil
	default:
		return gqltype, errors.New("resly: unsupported type " + gotype.String())
	}
}
