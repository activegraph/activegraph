package actionview

import (
	"fmt"

	"github.com/activegraph/activegraph/actioncontroller"
	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activesupport"
)

type ResultFunc func(*actioncontroller.Context) (interface{}, error)

func (fn ResultFunc) Execute(ctx *actioncontroller.Context) (interface{}, error) {
	return fn(ctx)
}

func Error(err error) actioncontroller.Result {
	return ResultFunc(func(ctx *actioncontroller.Context) (interface{}, error) {
		return nil, err
	})
}

func content(content interface{}) actioncontroller.Result {
	return ResultFunc(func(ctx *actioncontroller.Context) (interface{}, error) {
		switch content := content.(type) {
		case []activesupport.Hash:
			return content, nil
		case activesupport.HashConverter:
			return content.ToHash(), nil
		case activesupport.HashArrayConverter:
			return content.ToHashArray(), nil
		case nil:
			return nil, nil
		default:
			return nil, fmt.Errorf("%T does not support hash conversion", content)
		}
	})
}

func traverse(
	rec *activerecord.ActiveRecord,
	selection actioncontroller.QueryAttribute,
) (activesupport.Hash, error) {
	recHash := rec.ToHash()
	recHash = recHash.Slice(selection.NestedAttributeNames()...)

	for _, sel := range selection.NestedAttributes {
		if _, ok := recHash[sel.AttributeName]; ok {
			continue
		}

		target := rec.ReflectOnAssociation(sel.AttributeName)
		if target == nil {
			continue
		}

		switch target.Association.(type) {
		case activerecord.SingularAssociation:
			association, err := rec.AccessAssociation(sel.AttributeName)
			if err != nil {
				return nil, err
			}

			nestedHash, err := traverse(association, sel)
			if err != nil {
				return nil, err
			}
			recHash[sel.AttributeName] = nestedHash
		case activerecord.CollectionAssociation:
			collection, err := rec.AccessCollection(sel.AttributeName)
			if err != nil {
				return nil, err
			}
			associations, err := collection.ToA()
			if err != nil {
				return nil, err
			}

			nestedHash, err := traverseCollection(associations, sel)
			if err != nil {
				return nil, err
			}

			recHash[sel.AttributeName] = nestedHash
		default:
			panic("unknown target association")
		}
	}

	return recHash, nil
}

func traverseCollection(
	collection []*activerecord.ActiveRecord,
	selection actioncontroller.QueryAttribute,
) ([]activesupport.Hash, error) {
	collectionHash := make([]activesupport.Hash, 0, len(collection))

	for _, rec := range collection {
		recHash, err := traverse(rec, selection)
		if err != nil {
			return nil, err
		}
		collectionHash = append(collectionHash, recHash)
	}
	return collectionHash, nil
}

// NestedView returns a result with activesupport.Hash type.
//
// Method queries all nested attributes specified in ctx.Selection. That means
// additionall queries to a database are implied.
func NestedView(
	ctx *actioncontroller.Context, record activerecord.RecordResult,
) actioncontroller.Result {
	if record.IsErr() {
		return Error(record.Err())
	}
	if record.Ok().IsNone() {
		return content(nil)
	}

	result, err := traverse(
		record.Unwrap(), actioncontroller.QueryAttribute{
			NestedAttributes: ctx.Selection,
		},
	)
	if err != nil {
		Error(err)
	}
	return content(result)
}

// NestedCollectionView returns a colleciton as a slice of activesupport.Hash.
// Values of the record attributes are fetched as is, without conversion.
//
// In case of collection with error an error result is returned.
func NestedCollectionView(
	ctx *actioncontroller.Context,
	collection activerecord.CollectionResult,
) actioncontroller.Result {
	if collection.IsErr() {
		return Error(collection.Err())
	}
	if collection.Ok().IsNone() {
		return content(nil)
	}

	records, err := collection.Unwrap().ToA()
	if err != nil {
		return Error(err)
	}

	result, err := traverseCollection(
		records, actioncontroller.QueryAttribute{
			NestedAttributes: ctx.Selection,
		},
	)
	if err != nil {
		return Error(err)
	}
	return content(result)
}
