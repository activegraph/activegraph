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

func ViewResult(res activesupport.Res) actioncontroller.Result {
	return ResultFunc(func(ctx *actioncontroller.Context) (interface{}, error) {
		if res.Err() != nil {
			return nil, res.Err()
		}

		val, ok := res.Ok().(activesupport.HashConverter)
		if !ok {
			return nil, errors.Errorf("%T does not support hash conversion", val)
		}

		return val.ToHash(), nil
	})
}
