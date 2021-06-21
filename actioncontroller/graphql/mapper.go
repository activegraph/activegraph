package graphql

import (
	"net/http"
	"strings"

	"github.com/activegraph/activegraph/actioncontroller"
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
	model      actioncontroller.AbstractModel
	controller actioncontroller.AbstractController
}

type Mapper struct {
	resources []resource
}

func (m *Mapper) Resources(
	model actioncontroller.AbstractModel, controller actioncontroller.AbstractController,
) {
	m.resources = append(m.resources, resource{model, controller})
}

func (m *Mapper) primaryKey(model actioncontroller.AbstractModel) graphql.FieldConfigArgument {
	return graphql.FieldConfigArgument{
		model.PrimaryKey(): &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(
				typeconv(model.AttributeForInspect(model.PrimaryKey()).CastType()),
			),
		},
	}
}

func (m *Mapper) newIndexAction(
	model actioncontroller.AbstractModel, output graphql.Output, action actioncontroller.Action,
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
			action.Process(&actioncontroller.Context{
				Params: actioncontroller.Parameters(p.Args), Context: p.Context,
			})
			return nil, nil
		},
	}
}

func (m *Mapper) newShowAction(
	model actioncontroller.AbstractModel, output graphql.Output, action actioncontroller.Action,
) *graphql.Field {
	return &graphql.Field{
		Name: model.Name(),
		Args: m.primaryKey(model),
		Type: output,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			action.Process(&actioncontroller.Context{
				Params: actioncontroller.Parameters(p.Args), Context: p.Context,
			})
			return nil, nil
		},
	}
}

func (m *Mapper) newUpdateAction(
	operation string, model actioncontroller.AbstractModel, output graphql.Output, action actioncontroller.Action,
) *graphql.Field {

	objFields := make(graphql.InputObjectConfigFieldMap, len(action.ActionRequest()))
	for _, attr := range action.ActionRequest() {
		objFields[attr.AttributeName()] = &graphql.InputObjectFieldConfig{
			Type: typeconv(attr.CastType()),
		}
	}

	args := graphql.FieldConfigArgument{
		model.Name(): &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(graphql.NewInputObject(graphql.InputObjectConfig{
				Name:   strings.Title(operation) + strings.Title(model.Name()) + "Input",
				Fields: objFields,
			})),
		},
	}

	// TODO: separate creation and update
	if operation == "update" {
		args[model.PrimaryKey()] = m.primaryKey(model)[model.PrimaryKey()]
	}

	return &graphql.Field{
		Name: operation + strings.Title(model.Name()),
		Args: args,
		Type: output,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			context := &actioncontroller.Context{
				Context: p.Context,
				Params:  actioncontroller.Parameters(p.Args),
			}
			result := action.Process(context)
			return result.Execute(context)
		},
	}
}

func (m *Mapper) newDestroyAction(
	model actioncontroller.AbstractModel, output graphql.Output, action actioncontroller.Action,
) *graphql.Field {
	return &graphql.Field{
		Name: "delete" + strings.Title(model.Name()),
		Args: m.primaryKey(model),
		Type: output,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			action.Process(&actioncontroller.Context{
				Params: actioncontroller.Parameters(p.Args), Context: p.Context,
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
			case actioncontroller.ActionIndex:
				query := m.newIndexAction(resource.model, output, action)
				queries[query.Name] = query
			case actioncontroller.ActionShow:
				query := m.newShowAction(resource.model, output, action)
				queries[query.Name] = query
			case actioncontroller.ActionUpdate, actioncontroller.ActionCreate:
				mutation := m.newUpdateAction(action.ActionName(), resource.model, output, action)
				mutations[mutation.Name] = mutation
			case actioncontroller.ActionDestroy:
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
