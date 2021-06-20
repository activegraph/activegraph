package graphql

import (
	"net/http"
	"strings"

	"github.com/activegraph/activegraph/actiondispatch"
	"github.com/activegraph/activegraph/activerecord"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
)

func typeconv(t string) graphql.Type {
	switch t {
	case activerecord.Int:
		return graphql.Int
	case activerecord.String:
		return graphql.String
	default:
		return nil
	}
}

type resource struct {
	model      actiondispatch.AbstractModel
	controller actiondispatch.AbstractController
}

type Mapper struct {
	resources []resource
}

func (m *Mapper) Resources(
	model actiondispatch.AbstractModel, controller actiondispatch.AbstractController,
) {
	m.resources = append(m.resources, resource{model, controller})
}

func (m *Mapper) primaryKey(model actiondispatch.AbstractModel) graphql.FieldConfigArgument {
	return graphql.FieldConfigArgument{
		model.PrimaryKey(): &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(
				typeconv(model.AttributeForInspect(model.PrimaryKey()).CastType()),
			),
		},
	}
}

func (m *Mapper) newIndexAction(
	model actiondispatch.AbstractModel, output graphql.Output, action actiondispatch.Action,
) *graphql.Field {

	args := make(graphql.FieldConfigArgument, len(action.ActionRequest()))
	for _, attr := range action.ActionRequest() {
		args[attr.AttributeName()] = &graphql.ArgumentConfig{
			Type: typeconv(attr.CastType()),
		}
	}

	return &graphql.Field{
		Name: model.Name() + "s",
		Args: args,
		Type: graphql.NewList(output),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			action.Process(&actiondispatch.Context{
				Params: p.Args, Context: p.Context,
			})
			return nil, nil
		},
	}
}

func (m *Mapper) newShowAction(
	model actiondispatch.AbstractModel, output graphql.Output, action actiondispatch.Action,
) *graphql.Field {
	return &graphql.Field{
		Name: model.Name(),
		Args: m.primaryKey(model),
		Type: output,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			action.Process(&actiondispatch.Context{
				Params: p.Args, Context: p.Context,
			})
			return nil, nil
		},
	}
}

func (m *Mapper) newUpdateAction(
	operation string, model actiondispatch.AbstractModel, output graphql.Output, action actiondispatch.Action,
) *graphql.Field {

	objFields := make(graphql.InputObjectConfigFieldMap, len(action.ActionRequest()))
	for _, attr := range action.ActionRequest() {
		objFields[attr.AttributeName()] = &graphql.InputObjectFieldConfig{
			Type: typeconv(attr.CastType()),
		}
	}

	args := graphql.FieldConfigArgument{
		model.PrimaryKey(): m.primaryKey(model)[model.PrimaryKey()],
		model.Name(): &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(graphql.NewInputObject(graphql.InputObjectConfig{
				Name:   strings.Title(operation) + strings.Title(model.Name()) + "Input",
				Fields: objFields,
			})),
		},
	}

	return &graphql.Field{
		Name: operation + strings.Title(model.Name()),
		Args: args,
		Type: output,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			action.Process(&actiondispatch.Context{
				Params: p.Args, Context: p.Context,
			})
			return nil, nil
		},
	}
}

func (m *Mapper) newDestroyAction(
	model actiondispatch.AbstractModel, output graphql.Output, action actiondispatch.Action,
) *graphql.Field {
	return &graphql.Field{
		Name: "delete" + strings.Title(model.Name()),
		Args: m.primaryKey(model),
		Type: output,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			action.Process(&actiondispatch.Context{
				Params: p.Args, Context: p.Context,
			})
			return nil, nil
		},
	}
}

func (m *Mapper) Map() (http.Handler, error) {
	queries := make(graphql.Fields)
	mutations := make(graphql.Fields)

	for _, resource := range m.resources {
		objFields := make(graphql.Fields)
		for _, attrName := range resource.model.AttributeNames() {
			attr := resource.model.AttributeForInspect(attrName)

			objFields[attrName] = &graphql.Field{
				Name: attrName, Type: typeconv(attr.CastType()),
			}
		}

		output := graphql.NewObject(graphql.ObjectConfig{
			Name:   strings.Title(resource.model.Name()),
			Fields: objFields,
		})

		for _, action := range resource.controller.ActionMethods() {
			switch action.ActionName() {
			case actiondispatch.ActionIndex:
				query := m.newIndexAction(resource.model, output, action)
				queries[query.Name] = query
			case actiondispatch.ActionShow:
				query := m.newShowAction(resource.model, output, action)
				queries[query.Name] = query
			case actiondispatch.ActionUpdate, actiondispatch.ActionCreate:
				mutation := m.newUpdateAction(action.ActionName(), resource.model, output, action)
				mutations[mutation.Name] = mutation
			case actiondispatch.ActionDestroy:
				mutation := m.newDestroyAction(resource.model, output, action)
				mutations[mutation.Name] = mutation
			}
		}
	}

	var mutation *graphql.Object
	if len(mutations) > 0 {
		mutation = graphql.NewObject(graphql.ObjectConfig{
			Name: "Mutation", Fields: mutations,
		})
	}
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query", Fields: queries,
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: query, Mutation: mutation,
	})
	if err != nil {
		return nil, err
	}

	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	mux := http.NewServeMux()
	mux.Handle("/graphql", h)
	return mux, nil
}
