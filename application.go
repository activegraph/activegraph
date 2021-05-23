package activegraph

import (
	"net/http"
	"strings"

	"github.com/activegraph/activegraph/actioncontroller"
	"github.com/activegraph/activegraph/activerecord"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
)

type resource struct {
	rel        *activerecord.Relation
	controller *actioncontroller.ActionController
}

type A struct {
	resources []resource
}

func (a *A) Resources(
	rel *activerecord.Relation, controller *actioncontroller.ActionController,
) {
	a.resources = append(a.resources, resource{rel, controller})
}

type Application struct {
	resources []resource
}

func New(init func(*A)) *Application {
	app, err := Initialize(init)
	if err != nil {
		panic(err)
	}
	return app
}

func Initialize(init func(*A)) (*Application, error) {
	var a A
	init(&a)

	return &Application{resources: a.resources}, nil
}

func (a *Application) ListenAndServe() error {
	queries := make(graphql.Fields)

	for _, resource := range a.resources {
		objFields := make(graphql.Fields)
		for _, attrName := range resource.rel.AttributeNames() {
			attr := resource.rel.AttributeForInspect(attrName)
			var attrType graphql.Type
			switch attr.CastType() {
			case activerecord.Int:
				attrType = graphql.Int
			case activerecord.String:
				attrType = graphql.String
			}
			objFields[attrName] = &graphql.Field{Name: attrName, Type: attrType}
		}

		outputType := graphql.NewObject(graphql.ObjectConfig{
			Name:   strings.Title(resource.rel.Name()),
			Fields: objFields,
		})

		queries[resource.rel.Name()] = &graphql.Field{
			Name: resource.rel.Name(),
			Args: graphql.FieldConfigArgument{
				resource.rel.PrimaryKey(): &graphql.ArgumentConfig{Type: graphql.Int},
			},
			Type: outputType,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return nil, nil
			},
		}
	}

	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query", Fields: queries,
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{Query: query})
	if err != nil {
		return err
	}

	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	http.Handle("/graphql", h)
	return http.ListenAndServe(":8080", nil)
}
