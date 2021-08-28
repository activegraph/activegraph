package actioncontroller

import (
	"context"

	"github.com/activegraph/activegraph/activerecord"
)

type QueryAttribute struct {
	AttributeName    string
	NestedAttributes []QueryAttribute
}

func (qa QueryAttribute) NestedAttributeNames() []string {
	names := make([]string, len(qa.NestedAttributes))
	for i := range qa.NestedAttributes {
		names[i] = qa.NestedAttributes[i].AttributeName
	}
	return names
}

type Context struct {
	context.Context

	Params    Parameters
	Selection []QueryAttribute
}

// Result defines a contract that represents the result of action method.
type Result interface {
	Execute(*Context) (interface{}, error)
}

type Action interface {
	ActionName() string
	ActionRequest() []activerecord.Attribute
	Process(ctx *Context) Result
}

type ActionFunc func(*Context) Result

func (fn ActionFunc) Process(ctx *Context) Result {
	return fn(ctx)
}

type NamedAction struct {
	Name    string
	Request []activerecord.Attribute
	ActionFunc
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
