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

func queryconv(selections graphql.SelectionSet) []actioncontroller.QueryAttribute {
	attrs := make([]actioncontroller.QueryAttribute, 0, len(selections))

	// TODO: implement BFS selection flattening instead of recursion.
	for _, sel := range selections {
		switch sel := sel.(type) {
		case *graphql.Field:
			attr := actioncontroller.QueryAttribute{
				AttributeName: sel.Name,
			}

			if len(sel.SelectionSet) != 0 {
				attr.NestedAttributes = queryconv(sel.SelectionSet)
			}
			attrs = append(attrs, attr)
		case *graphql.FragmentSpread:
			panic("fragment spread is not implemented")
		case *graphql.InlineFragment:
			panic("inline fragment is not implemented")
		default:
			panic("unknown selection type")
		}
	}

	return attrs
}

func (rt *RoutingTable) Dispatch(r *Request, field *graphql.Field) (
	interface{}, error,
) {
	action, ok := rt.operations[field.Name]
	if !ok {
		return nil, actioncontroller.ErrActionNotFound{ActionName: field.Name}
	}

	// TODO: use StrongParameters to parse the arguments of a query?
	params := make(actioncontroller.Parameters, len(field.Arguments))
	for _, arg := range field.Arguments {
		// TODO: unmarshal the specified types.
		params[arg.Name] = arg.Value.Raw
	}

	ctx := &actioncontroller.Context{
		Context:   r.Context(),
		Params:    params,
		Selection: queryconv(field.SelectionSet),
	}
	spew.Dump(ctx.Selection)

	result := action.Process(ctx)
	return result.Execute(ctx)
}
