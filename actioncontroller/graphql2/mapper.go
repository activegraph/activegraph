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

func scalarconv(t activerecord.Type) *graphql.Type {
	switch t := t.(type) {
	case *activerecord.Int64:
		return &graphql.Type{NonNull: true, Elem: graphql.NamedType(Int.Name, nil)}
	case *activerecord.String:
		return &graphql.Type{NonNull: true, Elem: graphql.NamedType(String.Name, nil)}
	case *activerecord.DateTime:
		return &graphql.Type{NonNull: true, Elem: graphql.NamedType(DateTime.Name, nil)}
	case activerecord.Nil:
		return scalarconv(t.Type).Elem
	default:
		panic(t.String())
	}
}

type Schema struct {
	root *graphql.Schema
}

func (s *Schema) RegisterModel(model *activerecord.Relation) *graphql.Definition {
	queue := []*activerecord.Relation{model}
	originalName := strings.Title(model.Name())

	for len(queue) != 0 {
		model = queue[0]
		queue = queue[1:]

		// Ensure the model is not registered yet with this name.
		name := strings.Title(model.Name())
		if _, ok := s.root.Types[name]; ok {
			continue
		}

		attrs := model.AttributesForInspect()
		assocs := model.ReflectOnAllAssociations()

		fields := make(graphql.FieldList, 0, len(attrs)+len(assocs))

		for _, attr := range attrs {
			fields = append(fields, &graphql.FieldDefinition{
				Name: attr.AttributeName(),
				Type: scalarconv(attr.AttributeType()),
			})
		}

		for _, assoc := range assocs {
			// Put a type dependency to the queue of registration.
			queue = append(queue, assoc.Relation)
			assocName := strings.Title(assoc.AssociationName())

			fields = append(fields, &graphql.FieldDefinition{
				Name: assoc.AssociationName(),
				// TODO: what about modifications (non-nil/list) ?
				Type: graphql.NamedType(assocName, nil),
			})
		}

		// Register a new object type.
		s.root.Types[name] = &graphql.Definition{
			Kind:       graphql.Object,
			Name:       name,
			Interfaces: make([]string, 0),
			Fields:     fields,
		}
	}

	return s.root.Types[originalName]
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
	schema := &graphql.Schema{
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

	rootSchema := Schema{schema}

	for _, resource := range m.resources {
		outputType := rootSchema.RegisterModel(resource.model.(*activerecord.Relation))

		for _, action := range resource.controller.ActionMethods() {
			switch action.ActionName() {
			case actioncontroller.ActionIndex:
				schema.Query.Fields = append(schema.Query.Fields, &graphql.FieldDefinition{
					Name: resource.model.Name() + "s",
					Type: &graphql.Type{
						Elem: &graphql.Type{
							NonNull: true,
							Elem:    graphql.NamedType(outputType.Name, nil),
						},
					},
				})
			default:
				fmt.Printf("action %q is not supported\n", action.ActionName())
			}
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/graphql", NewHandler(HandlerFunc(DefaultHandler), schema))
	return mux, nil
}
