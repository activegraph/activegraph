package activegraph

import (
	"net/http"

	"github.com/activegraph/activegraph/actioncontroller"
	"github.com/activegraph/activegraph/actioncontroller/graphql2"
)

type A struct {
	actioncontroller.Mapper
}

type Application struct {
	mapper actioncontroller.Mapper
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

	return http.ListenAndServe(":3000", handler)
}
