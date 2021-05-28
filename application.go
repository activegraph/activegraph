package activegraph

import (
	"net/http"

	"github.com/activegraph/activegraph/actiondispatch"
	"github.com/activegraph/activegraph/actiondispatch/graphql"
)

type A struct {
	actiondispatch.Mapper
}

type Application struct {
	mapper actiondispatch.Mapper
}

func New(init func(*A)) *Application {
	app, err := Initialize(init)
	if err != nil {
		panic(err)
	}
	return app
}

func Initialize(init func(*A)) (*Application, error) {
	a := A{Mapper: new(graphql.Mapper)}
	init(&a)

	return &Application{mapper: a.Mapper}, nil
}

func (a *Application) ListenAndServe() error {
	handler, err := a.mapper.Map()
	if err != nil {
		return err
	}

	http.Handle("/graphql", handler)
	return http.ListenAndServe(":8080", nil)
}
