package graphql

import (
	"fmt"
	"strconv"
	"time"

	"github.com/davecgh/go-spew/spew"
	graphql "github.com/vektah/gqlparser/v2/ast"

	"github.com/activegraph/activegraph/actioncontroller"
	"github.com/activegraph/activegraph/activesupport"
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

func unmarshalArg(v *graphql.Value) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	switch v.Kind {
	case graphql.IntValue:
		return strconv.ParseInt(v.Raw, 10, 64)
	case graphql.FloatValue:
		return strconv.ParseFloat(v.Raw, 64)
	case graphql.StringValue, graphql.BlockValue, graphql.EnumValue:
		// TODO: Implement a normal solution, not this.
		if v.ExpectedType.Name() == "DateTime" {
			return time.Parse("2006-01-02 15:04:05.000000 MST", v.Raw)
		}
		return v.Raw, nil
	case graphql.BooleanValue:
		return strconv.ParseBool(v.Raw)
	case graphql.NullValue:
		return nil, nil
	case graphql.ListValue:
		var val []interface{}
		for _, elem := range v.Children {
			elemVal, err := unmarshalArg(elem.Value)
			if err != nil {
				return val, err
			}
			val = append(val, elemVal)
		}
		return val, nil
	case graphql.ObjectValue:
		val := activesupport.Hash{}
		for _, elem := range v.Children {
			elemVal, err := unmarshalArg(elem.Value)
			if err != nil {
				return val, err
			}
			val[elem.Name] = elemVal
		}
		return val, nil
	default:
		panic(fmt.Errorf("unknown value kind %d", v.Kind))
	}
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
		value, err := unmarshalArg(arg.Value)
		if err != nil {
			return nil, err
		}
		params[arg.Name] = value
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
