package actionview

import (
	"fmt"
	"strings"

	"github.com/activegraph/activegraph/actioncontroller"
	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activesupport"

	"github.com/pkg/errors"
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
) (interface{}, error) {

	// TODO: do not trim suffix here!
	target := rec.ReflectOnAssociation(selection.AttributeName)
	fmt.Println("!!!", rec, strings.TrimSuffix(selection.AttributeName, "s"))
	fmt.Printf("\tnested: %s / %v || %v\n", rec, selection, target)
	if target == nil {
		return nil, nil
	}

	switch target.Association.(type) {
	case activerecord.SingularAssociation:
		assoc, err := rec.AccessAssociation(selection.AttributeName)
		if assoc == nil || err != nil {
			return nil, err
		}

		// TODO: Slice hash to take only necessary attributes.
		assocHash := assoc.ToHash()
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
	case activerecord.CollectionAssociation:
		assocs, err := rec.AccessCollection(selection.AttributeName)
		if assocs == nil || err != nil {
			return nil, err
		}

		result := make([]activesupport.Hash, 0, len(assocs))
		for _, assoc := range assocs {
			// TODO: Slice hash to take only necessary attributes.
			assocHash := assoc.ToHash()
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
			result = append(result, assocHash)
		}
		return result, nil
	default:
		// TODO: replace with an error.
		panic("unknown target association")
	}
}

func GraphResult(ctx *actioncontroller.Context, res activerecord.Result) actioncontroller.Result {
	if res.IsErr() {
		return ContentResult(res)
	}

	record := res.UnwrapRecord()
	recordHash := make(activesupport.Hash)

	for _, sel := range ctx.Selection {
		if record.HasAttribute(sel.AttributeName) {
			recordHash[sel.AttributeName] = record.Attribute(sel.AttributeName)
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
