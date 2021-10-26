package graphql

import (
	"github.com/davecgh/go-spew/spew"
	graphql "github.com/vektah/gqlparser/v2/ast"

	"github.com/activegraph/activegraph/actioncontroller"
)

type RoutingTable struct {
	operations map[string]actioncontroller.Action
}

func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		operations: make(map[string]actioncontroller.Action),
	}
}

func (rt *RoutingTable) AddOperation(name string, action actioncontroller.Action) {
	rt.operations[name] = action
}

func (rt *RoutingTable) Dispatch(r *Request, field *graphql.Field) (
	interface{}, error,
) {
	spew.Dump(field)
	action, ok := rt.operations[field.Name]
	if !ok {
		return nil, actioncontroller.ErrActionNotFound{ActionName: field.Name}
	}

	ctx := &actioncontroller.Context{
		Context: r.Context(),
	}

	result := action.Process(ctx)
	return result.Execute(ctx)
}
