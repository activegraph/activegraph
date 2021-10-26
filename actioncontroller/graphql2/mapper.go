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

func CanonicalModelName(modelName string) string {
	return strings.Title(modelName)
}

type Schema struct {
	root *graphql.Schema
}

func (s *Schema) AddIndexOp(model *activerecord.Relation) *graphql.FieldDefinition {
	def := &graphql.FieldDefinition{
		Name: model.Name() + "s",
		Type: &graphql.Type{
			Elem: &graphql.Type{
				NonNull: true,
				Elem:    graphql.NamedType(CanonicalModelName(model.Name()), nil),
			},
		},
	}

	s.root.Query.Fields = append(s.root.Query.Fields, def)
	return def
}

func (s *Schema) AddShowOp(model *activerecord.Relation) *graphql.FieldDefinition {
	def := &graphql.FieldDefinition{
		Name: model.Name(),
		Arguments: graphql.ArgumentDefinitionList{
			{
				Name: model.PrimaryKey(),
				Type: scalarconv(model.AttributeForInspect(model.PrimaryKey()).AttributeType()),
			},
		},
		Type: graphql.NamedType(CanonicalModelName(model.Name()), nil),
	}

	s.root.Query.Fields = append(s.root.Query.Fields, def)
	return def
}

func (s *Schema) AddCreateOp(
	model *activerecord.Relation, action actioncontroller.Action,
) *graphql.FieldDefinition {
	inputs := action.ActionRequest()
	args := make(graphql.ArgumentDefinitionList, 0, len(inputs))

	for _, input := range inputs {
		// TODO: add support of objects.
		args = append(args, &graphql.ArgumentDefinition{
			Name: input.AttributeName(),
			Type: scalarconv(input.AttributeType()),
		})
	}

	def := &graphql.FieldDefinition{
		Name:      "create" + CanonicalModelName(model.Name()),
		Arguments: args,
		Type:      graphql.NamedType(CanonicalModelName(model.Name()), nil),
	}

	s.root.Mutation.Fields = append(s.root.Mutation.Fields, def)
	return def
}

func (s *Schema) AddDestroyOp(model *activerecord.Relation) *graphql.FieldDefinition {
	def := &graphql.FieldDefinition{
		Name: "delete" + CanonicalModelName(model.Name()),
		Arguments: graphql.ArgumentDefinitionList{
			{
				Name: model.PrimaryKey(),
				Type: scalarconv(model.AttributeForInspect(model.PrimaryKey()).AttributeType()),
			},
		},
		Type: graphql.NamedType(CanonicalModelName(model.Name()), nil),
	}

	s.root.Mutation.Fields = append(s.root.Mutation.Fields, def)
	return def
}

func (s *Schema) AddModel(model *activerecord.Relation) *graphql.Definition {
	queue := []*activerecord.Relation{model}
	canonicalName := CanonicalModelName(model.Name())

	for len(queue) != 0 {
		model = queue[0]
		queue = queue[1:]

		// Ensure the model is not registered yet with this name.
		name := CanonicalModelName(model.Name())
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

	return s.root.Types[canonicalName]
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
	routing := NewRoutingTable()

	for _, resource := range m.resources {
		model := resource.model.(*activerecord.Relation)
		rootSchema.AddModel(model)

		for _, action := range resource.controller.ActionMethods() {
			switch action.ActionName() {
			case actioncontroller.ActionShow:
				op := rootSchema.AddShowOp(model)
				routing.AddOperation(op.Name, action)
			case actioncontroller.ActionIndex:
				op := rootSchema.AddIndexOp(model)
				routing.AddOperation(op.Name, action)
			case actioncontroller.ActionCreate:
				op := rootSchema.AddCreateOp(model, action)
				routing.AddOperation(op.Name, action)
			case actioncontroller.ActionDestroy:
				op := rootSchema.AddDestroyOp(model)
				routing.AddOperation(op.Name, action)
			default:
				fmt.Printf("action %q is not supported\n", action.ActionName())
			}
		}
	}

	handler := func(rw ResponseWriter, r *Request) {
		if r.query.Operations.ForName("IntrospectionQuery") != nil {
			IntrospectionHandler(rw, r)
			return
		}

		for _, op := range r.query.Operations {
			for _, selection := range op.SelectionSet {
				field := selection.(*graphql.Field)
				data, err := routing.Dispatch(r, field)

				rw.WriteError(err)
				rw.WriteData(field.Name, data)
			}
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/graphql", NewHandler(HandlerFunc(handler), schema))
	return mux, nil
}
