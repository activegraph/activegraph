package actiondispatch

import (
	"context"

	"github.com/activegraph/activegraph/activerecord"
)

type Context struct {
	context.Context

	Params map[string]interface{}
}

type Action interface {
	ActionName() string
	ActionRequest() []activerecord.Attribute
	Process(ctx *Context) error
}

type AnonymousAction func(*Context) error

func (fn AnonymousAction) Process(ctx *Context) error {
	return fn(ctx)
}

type NamedAction struct {
	Name    string
	Request []activerecord.Attribute
	AnonymousAction
}

func (a *NamedAction) ActionRequest() []activerecord.Attribute {
	return a.Request
}

func (a *NamedAction) ActionName() string {
	return a.Name
}

const (
	ActionCreate  = "create"
	ActionUpdate  = "update"
	ActionShow    = "show"
	ActionIndex   = "index"
	ActionDestroy = "destroy"
)

func IsCanonicalAction(actionName string) bool {
	switch actionName {
	case ActionCreate, ActionUpdate, ActionShow, ActionIndex, ActionDestroy:
		return true
	default:
		return false
	}
}
