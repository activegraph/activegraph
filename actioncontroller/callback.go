package actioncontroller

import (
	"github.com/activegraph/activegraph/activesupport"
)

type Callback func(ctx *Context) Result

type CallbackAround func(ctx *Context, action Action) Result

type callback struct {
	fn   Callback
	only activesupport.Hash
}

func (cb *callback) run(ctx *Context, action Action) Result {
	if cb.only.IsEmpty() || cb.only.HasKey(action.ActionName()) {
		return cb.fn(ctx)
	}
	return nil
}

type callbackAround struct {
	fn   CallbackAround
	only activesupport.Hash
}

func (cb *callbackAround) run(ctx *Context, action Action) Result {
	if cb.only.IsEmpty() || cb.only.HasKey(action.ActionName()) {
		return cb.fn(ctx, action)
	}
	return action.Process(ctx)
}

func (cb *callbackAround) chain(action Action) Action {
	return &NamedAction{
		Name:        action.ActionName(),
		Constraints: action.ActionConstraints(),
		ActionFunc:  func(ctx *Context) Result { return cb.run(ctx, action) },
	}
}

type callbacks struct {
	before []callback
	after  []callback
	around []callbackAround
}

func (cbs *callbacks) appendBeforeAction(cb Callback, only []string) {
	if cb == nil {
		panic("nil callback")
	}
	cbs.before = append(cbs.before, callback{cb, activesupport.StringSlice(only).ToHash()})
}

func (cbs *callbacks) appendAfterAction(cb Callback, only []string) {
	if cb == nil {
		panic("nil callback")
	}
	cbs.after = append(cbs.after, callback{cb, activesupport.StringSlice(only).ToHash()})
}

func (cbs *callbacks) appendAroundAction(cb CallbackAround, only []string) {
	if cb == nil {
		panic("nil callback")
	}
	cbs.around = append(cbs.around, callbackAround{cb, activesupport.StringSlice(only).ToHash()})
}

func (cbs *callbacks) runCallbacks(ctx *Context, action Action) (result Result) {
	for _, cb := range cbs.before {
		res := cb.run(ctx, action)
		if res != nil {
			return res
		}
	}

	for _, cb := range cbs.around {
		action = cb.chain(action)
	}

	result = action.Process(ctx)

	for _, cb := range cbs.after {
		res := cb.run(ctx, action)
		if res != nil {
			return res
		}
	}
	return result
}

// TODO: Return `Action` instead of `NamedAction`.
func (cbs *callbacks) override(action Action) *NamedAction {
	return &NamedAction{
		Name:        action.ActionName(),
		Constraints: action.ActionConstraints(),
		ActionFunc:  func(ctx *Context) Result { return cbs.runCallbacks(ctx, action) },
	}
}
