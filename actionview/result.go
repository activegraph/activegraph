package actionview

import (
	"github.com/pkg/errors"

	"github.com/activegraph/activegraph/actioncontroller"
	"github.com/activegraph/activegraph/activesupport"
)

type ResultFunc func(*actioncontroller.Context) (interface{}, error)

func (fn ResultFunc) Execute(ctx *actioncontroller.Context) (interface{}, error) {
	return fn(ctx)
}

func ViewResult(res activesupport.Result) actioncontroller.Result {
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
