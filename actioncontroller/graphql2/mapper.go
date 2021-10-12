package graphql

import (
	"fmt"
	"net/http"
	"strings"

	graphql "github.com/vektah/gqlparser/v2/ast"

	"github.com/activegraph/activegraph/actioncontroller"
	"github.com/activegraph/activegraph/activerecord"
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

func scalarconv(t activerecord.Type) string {
	switch t := t.(type) {
	case *activerecord.Int64:
		return Int.Name
	case *activerecord.String:
		return String.Name
	case *activerecord.DateTime:
		return DateTime.Name
	case activerecord.Nil:
		return scalarconv(t.Type)
	default:
		panic(t.String())
	}
}

func objconv(name string, model *activerecord.Relation) *graphql.Definition {
	attrs := model.AttributesForInspect()
	fields := make(graphql.FieldList, 0, len(attrs))

	for _, attr := range attrs {
		fields = append(fields, &graphql.FieldDefinition{
			Name: attr.AttributeName(),
			Type: &graphql.Type{NamedType: scalarconv(attr.AttributeType())},
		})
	}

	// for _, assoc := range model.ReflectOnAllAssociations() {
	// 	fields = append(fields, &graphql.FieldDefinition{
	// 		Name: assoc.AssociationName(),
	// 		Type: &graphql.Type{NamedType: assoc.AssociationName()},
	// 	})
	// }

	return &graphql.Definition{
		Kind:       graphql.Object,
		Name:       name,
		Interfaces: make([]string, 0),
		Fields:     fields,
	}
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

func (m *Mapper) Map() (http.Handler, error) {
	schema := graphql.Schema{
		Query: &graphql.Definition{
			Name: "Query",
			Kind: graphql.Object,
		},
		Mutation: &graphql.Definition{
			Name: "Mutation",
			Kind: graphql.Object,
		},
		Types: make(map[string]*graphql.Definition),
	}

	schema.Types["Query"] = schema.Query
	schema.Types["Mutation"] = schema.Mutation

	schema.Types["Int"] = Int
	schema.Types["String"] = String
	schema.Types["DateTime"] = DateTime

	for _, resource := range m.resources {
		outputType := objconv(
			strings.Title(resource.model.Name()),
			resource.model.(*activerecord.Relation),
		)
		schema.Types[outputType.Name] = outputType

		for _, action := range resource.controller.ActionMethods() {
			switch action.ActionName() {
			case actioncontroller.ActionIndex:
				schema.Query.Fields = append(schema.Query.Fields, &graphql.FieldDefinition{
					Name: resource.model.Name() + "s",
					Type: graphql.ListType(graphql.NamedType(outputType.Name, nil), nil),
				})
			default:
				fmt.Printf("action %q is not supported\n", action.ActionName())
			}
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/graphql", NewHandler(HandlerFunc(DefaultHandler), &schema))
	return mux, nil
}
