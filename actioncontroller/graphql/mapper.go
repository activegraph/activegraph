package graphql

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/activegraph/activegraph/actioncontroller"
	"github.com/activegraph/activegraph/activerecord"
	"github.com/graphql-go/graphql/language/ast"

	"github.com/graphql-go/graphql"
)

type ErrConstraintNotFound struct {
	Operation  string
	Name       string
	Constraint string
}

func (e ErrConstraintNotFound) Error() string {
	return fmt.Sprintf(
		"%s constraint for %s '%s' not found", e.Constraint, e.Operation, e.Name,
	)
}

const (
	// GraphQL operations.
	OperationQuery        = "query"        // a read-only fetch.
	OperationMutation     = "mutation"     // a write followed by fetch.
	OperationSubscription = "subscription" // unsupported yet.
	OperationUnknown      = ""
)

func typeconv(t activerecord.Type) graphql.Type {
	switch t := t.(type) {
	case *activerecord.Int64:
		return graphql.NewNonNull(graphql.Int)
	case *activerecord.String:
		return graphql.NewNonNull(graphql.String)
	case *activerecord.Boolean:
		return graphql.NewNonNull(graphql.Boolean)
	case *activerecord.Float64:
		return graphql.NewNonNull(graphql.Float)
	case activerecord.Nil:
		if gt := typeconv(t.Type); gt != nil {
			return graphql.GetNullable(gt).(graphql.Type)
		}
	}
	return nil
}

func argsconv(attrs []activerecord.Attribute) graphql.FieldConfigArgument {
	args := make(graphql.FieldConfigArgument, len(attrs))
	for _, attr := range attrs {
		args[attr.AttributeName()] = &graphql.ArgumentConfig{
			Type: typeconv(attr.AttributeType()),
		}
	}
	return args
}

func payloadconv(name string, attrs []activerecord.Attribute) *graphql.Object {
	fields := make(graphql.Fields, len(attrs))
	for _, attr := range attrs {
		fields[attr.AttributeName()] = &graphql.Field{
			Name: attr.AttributeName(), Type: typeconv(attr.AttributeType()),
		}
	}
	return graphql.NewObject(graphql.ObjectConfig{Name: name, Fields: fields})
}

func objconv(name string, model *activerecord.Relation, viewed map[string]struct{}) *graphql.Object {
	attrs := model.AttributesForInspect()
	fields := make(graphql.Fields, len(attrs))

	for _, attr := range attrs {
		fields[attr.AttributeName()] = &graphql.Field{
			Name: attr.AttributeName(), Type: typeconv(attr.AttributeType()),
		}
	}

	viewed[name] = struct{}{}
	for _, assoc := range model.ReflectOnAllAssociations() {
		if _, ok := viewed[strings.Title(assoc.AssociationName())]; ok {
			continue
		}
		fields[assoc.AssociationName()] = &graphql.Field{
			Name: assoc.AssociationName(),
			Type: objconv(strings.Title(assoc.AssociationName()), assoc.Relation, viewed),
		}
	}

	return graphql.NewObject(graphql.ObjectConfig{Name: name, Fields: fields})
}

type resource struct {
	model      actioncontroller.AbstractModel
	controller actioncontroller.AbstractController
}

type matching struct {
	operation   string
	name        string
	action      actioncontroller.Action
	constraints actioncontroller.Constraints
}

func queryconv(selections []ast.Selection) []actioncontroller.QueryAttribute {
	attrs := make([]actioncontroller.QueryAttribute, 0, len(selections))
	for _, sel := range selections {
		field, ok := sel.(*ast.Field)
		if !ok {
			continue
		}

		attr := actioncontroller.QueryAttribute{
			AttributeName: field.Name.Value,
		}
		if field.SelectionSet != nil {
			attr.NestedAttributes = queryconv(field.SelectionSet.Selections)
		}
		attrs = append(attrs, attr)
	}
	return attrs
}

func newResolveFunc(action actioncontroller.Action) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		context := &actioncontroller.Context{
			Context: p.Context,
			Params:  actioncontroller.Parameters(p.Args),
		}

		if len(p.Info.FieldASTs) > 0 {
			selections := p.Info.FieldASTs[0].SelectionSet.Selections
			context.Selection = queryconv(selections)
		}

		result := action.Process(context)
		return result.Execute(context)
	}
}

type Mapper struct {
	resources []resource
	matchings []matching
}

func (m *Mapper) Resources(
	model actioncontroller.AbstractModel, controller actioncontroller.AbstractController,
) {
	m.resources = append(m.resources, resource{model, controller})
}

func (m *Mapper) Match(
	via, path string,
	action actioncontroller.Action,
	constraints ...actioncontroller.Constraints,
) {
	var constraint actioncontroller.Constraints
	if len(constraints) > 0 {
		constraint = constraints[len(constraints)-1]
	}
	if constraint.Request == nil {
		panic(ErrConstraintNotFound{Name: path, Operation: via, Constraint: "request"})
	}
	if constraint.Response == nil {
		panic(ErrConstraintNotFound{Name: path, Operation: via, Constraint: "response"})
	}

	m.matchings = append(m.matchings, matching{via, path, action, constraint})
}

func (m *Mapper) primaryKey(model actioncontroller.AbstractModel) graphql.FieldConfigArgument {
	return graphql.FieldConfigArgument{
		model.PrimaryKey(): &graphql.ArgumentConfig{
			Type: typeconv(model.AttributeForInspect(model.PrimaryKey()).AttributeType()),
		},
	}
}

func (m *Mapper) newAction(
	name string,
	args []activerecord.Attribute,
	result []activerecord.Attribute,
	action actioncontroller.Action,
) *graphql.Field {
	return &graphql.Field{
		Name:    name,
		Args:    argsconv(args),
		Type:    payloadconv(strings.Title(name)+"Payload", result),
		Resolve: newResolveFunc(action),
	}
}

func (m *Mapper) newIndexAction(
	model actioncontroller.AbstractModel, output graphql.Output, action actioncontroller.Action,
) *graphql.Field {

	args := make(graphql.FieldConfigArgument, len(action.ActionRequest()))
	for _, attr := range action.ActionRequest() {
		args[attr.AttributeName()] = &graphql.ArgumentConfig{
			Type: typeconv(attr.AttributeType()),
		}
	}

	return &graphql.Field{
		Name:    model.Name() + "s",
		Args:    args,
		Type:    graphql.NewList(output),
		Resolve: newResolveFunc(action),
	}
}

func (m *Mapper) newShowAction(
	model actioncontroller.AbstractModel, output graphql.Output, action actioncontroller.Action,
) *graphql.Field {
	return &graphql.Field{
		Name:    model.Name(),
		Args:    m.primaryKey(model),
		Type:    output,
		Resolve: newResolveFunc(action),
	}
}

func (m *Mapper) newUpdateAction(
	operation string, model actioncontroller.AbstractModel, output graphql.Output, action actioncontroller.Action,
) *graphql.Field {

	objFields := make(graphql.InputObjectConfigFieldMap, len(action.ActionRequest()))
	for _, attr := range action.ActionRequest() {
		objFields[attr.AttributeName()] = &graphql.InputObjectFieldConfig{
			Type: typeconv(attr.AttributeType()),
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
		Name:    operation + strings.Title(model.Name()),
		Args:    args,
		Type:    output,
		Resolve: newResolveFunc(action),
	}
}

func (m *Mapper) newDestroyAction(
	model actioncontroller.AbstractModel, output graphql.Output, action actioncontroller.Action,
) *graphql.Field {
	return &graphql.Field{
		Name:    "delete" + strings.Title(model.Name()),
		Args:    m.primaryKey(model),
		Type:    output,
		Resolve: newResolveFunc(action),
	}
}

func (m *Mapper) Map() (http.Handler, error) {
	queries := make(graphql.Fields)
	mutations := make(graphql.Fields)

	for _, resource := range m.resources {
		output := objconv(
			strings.Title(resource.model.Name()),
			resource.model.(*activerecord.Relation),
			// TODO: rework without recursion.
			make(map[string]struct{}),
		)

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
			default:
				// println("consider registering non-canonical action?")
			}
		}
	}

	for _, matching := range m.matchings {
		switch matching.operation {
		case OperationQuery:
		case OperationMutation:
			mutations[matching.name] = m.newAction(
				matching.name,
				matching.constraints.Request.Attributes,
				matching.constraints.Response.Attributes,
				matching.action,
			)
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

	mux := http.NewServeMux()
	mux.Handle("/graphql", GraphQLHandler(schema))
	return mux, nil
}
