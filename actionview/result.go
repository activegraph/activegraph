package actionview

import (
	"fmt"
	"github.com/pkg/errors"

	"github.com/activegraph/activegraph/actioncontroller"
	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activesupport"
)

type ResultFunc func(*actioncontroller.Context) (interface{}, error)

func (fn ResultFunc) Execute(ctx *actioncontroller.Context) (interface{}, error) {
	return fn(ctx)
}

func ContentResult(res activesupport.Result) actioncontroller.Result {
	return ResultFunc(func(ctx *actioncontroller.Context) (interface{}, error) {
		if res.IsErr() {
			return nil, res.Err()
		}

		switch val := res.Ok().(type) {
		case activesupport.HashConverter:
			return val.ToHash(), nil
		case activesupport.HashArrayConverter:
			return val.ToHashArray(), nil
		default:
			return nil, errors.Errorf("%T does not support hash conversion", val)
		}
	})
}

func queryNested(
	rec *activerecord.ActiveRecord,
	selection actioncontroller.QueryAttribute,
) (activesupport.Hash, error) {
	fmt.Printf("\tnested: %s / %v\n", rec, selection)
	assoc, err := rec.AccessAssociation(selection.AttributeName)
	if err != nil {
		return nil, err
	}

	assocHash := assoc.ToHash()
	if len(selection.NestedAttributes) == 0 {
		return assocHash, nil
	}

	for _, sel := range selection.NestedAttributes {
		if _, ok := assocHash[sel.AttributeName]; ok {
			continue
		}

		selectionHash, err := queryNested(assoc, sel)
		if err != nil {
			return nil, err
		}
		assocHash[sel.AttributeName] = selectionHash
	}
	return assocHash, nil
}

func GraphResult(ctx *actioncontroller.Context, res activerecord.Result) actioncontroller.Result {
	fmt.Println("!!!", res)
	if res.IsErr() {
		return ContentResult(res)
	}

	record := res.UnwrapRecord()
	recordHash := record.ToHash()
	for _, sel := range ctx.Selection {
		if _, ok := recordHash[sel.AttributeName]; ok {
			continue
		}

		selectionHash, err := queryNested(record, sel)
		if err != nil {
			return ContentResult(activesupport.Err(err))
		}
		recordHash[sel.AttributeName] = selectionHash
	}

	return ContentResult(activesupport.Ok(recordHash))
}
