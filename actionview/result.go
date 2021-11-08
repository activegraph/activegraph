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

func Content(content interface{}) actioncontroller.Result {
	return ResultFunc(func(ctx *actioncontroller.Context) (interface{}, error) {
		switch content := content.(type) {
		case []activesupport.Hash:
			return content, nil
		case activesupport.HashConverter:
			return content.ToHash(), nil
		case activesupport.HashArrayConverter:
			return content.ToHashArray(), nil
		case error:
			return nil, content
		case nil:
			return nil, nil
		default:
			return nil, errors.Errorf("%T does not support hash conversion", content)
		}
	})
}

func queryNested(
	rec *activerecord.ActiveRecord,
	selection actioncontroller.QueryAttribute,
) (interface{}, error) {

	target := rec.ReflectOnAssociation(selection.AttributeName)
	fmt.Println("!!!", rec, strings.TrimSuffix(selection.AttributeName, "s"))
	fmt.Printf("\tnested: %s / %v || %v\n", rec, selection, target)
	if target == nil {
		return nil, nil
	}

	switch target.Association.(type) {
	case activerecord.SingularAssociation:
		assoc, err := rec.AccessAssociation(selection.AttributeName)
		if err != nil {
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
		collection, err := rec.AccessCollection(selection.AttributeName)
		if err != nil {
			return nil, err
		}

		assocs, err := collection.ToA()
		if err != nil {
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

func Graph(ctx *actioncontroller.Context, res activesupport.Result) actioncontroller.Result {
	if res.IsErr() {
		return Content(res.Err())
	} else if res.Ok().IsNone() {
		return Content(nil)
	}

	//option := res.Ok().Unwrap()
	switch option := res.Ok().Unwrap().(type) {
	case activerecord.Option:
		record := option.Unwrap()
		recordHash := make(activesupport.Hash)

		for _, sel := range ctx.Selection {
			if record.HasAttribute(sel.AttributeName) {
				recordHash[sel.AttributeName] = record.Attribute(sel.AttributeName)
				continue
			}

			selectionHash, err := queryNested(record, sel)
			if err != nil {
				return Content(err)
			}
			recordHash[sel.AttributeName] = selectionHash
		}

		return Content(recordHash)
	case activerecord.CollectionOption:
		records, err := option.Unwrap().ToA()
		if err != nil {
			return Content(err)
		}

		result := make([]activesupport.Hash, 0, len(records))
		for _, record := range records {
			recordHash := make(activesupport.Hash)
			for _, sel := range ctx.Selection {
				if record.HasAttribute(sel.AttributeName) {
					recordHash[sel.AttributeName] = record.Attribute(sel.AttributeName)
					continue
				}

				selectionHash, err := queryNested(record, sel)
				if err != nil {
					return Content(err)
				}
				recordHash[sel.AttributeName] = selectionHash
			}
			result = append(result, recordHash)
		}
		return Content(result)
	default:
		panic("unsupported result")
	}
}
