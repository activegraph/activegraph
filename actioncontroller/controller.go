package actioncontroller

import (
	"fmt"

	"github.com/activegraph/activegraph/activerecord"
)

// ErrActionNotFound is returned when a non-existing action is triggered.
type ErrActionNotFound struct {
	ActionName string
}

// Error returns a string representation of the error.
func (e ErrActionNotFound) Error() string {
	return fmt.Sprintf("action %q not found", e.ActionName)
}

type actionsMap map[string]*NamedAction

func (m actionsMap) copy() actionsMap {
	mm := make(actionsMap, len(m))
	for name, action := range m {
		mm[name] = action
	}
	return mm
}

type C struct {
	actions   actionsMap
	callbacks callbacks

	params map[string][]activerecord.Attribute
}

// AppendBeforeAction appends a callback before actions. See Callback for parameter
// details.
//
// Use the before callback to execute necessary logic before executing an action, the
// implementation could completely override the behavior of the action.
//
// The chain of "before" callbacks execution interrupts when the non-nil result
// is returned. It's anticipated that before filters are often used to
// prevent from execution certain operations or queries.
//
//	AdminController := actioncontroller.New(func(c *actioncontroller.C) {
//		c.AppendBeforeAction(func(ctx *actioncontroller.Context) actioncontroller.Result {
//			if !isAuthorized(ctx) {
//				return actionview.ContentResult(nil, errors.New("forbidden"))
//			}
//			return nil
//		})
//	})
func (c *C) AppendBeforeAction(cb Callback, only ...string) {
	c.callbacks.appendBeforeAction(cb, only)
}

func (c *C) BeforeAction(cb Callback, only ...string) {
	c.AppendBeforeAction(cb, only...)
}

func (c *C) AppendAfterAction(cb Callback, only ...string) {
	c.callbacks.appendAfterAction(cb, only)
}

func (c *C) AfterAction(cb Callback, only ...string) {
	c.AppendAfterAction(cb, only...)
}

// AppendAroundAction appends a callback around action. See CallbackAround for parameter
// details.
//
// Use the around callback to wrap the action with extra logic, e.g. execute
// all operations within an action in a database transaction.
//
//	func WrapInTransaction(
//		ctx *actioncontroller.Context, action actioncontroller.Action
//	) (result actioncontroller.Result) {
//		err := activerecord.Transaction(ctx, func() error {
//			result = action.Process(ctx)
//			return result.Err()
//		})
//		if err != nil {
//			return actionview.ContentResult(nil, err)
//		}
//		return nil
//	}
//
//	ProductsController := actioncontroller.New(func(c *actioncontroller.C) {
//		c.AppendAroundAction(WrapInTransaction)
//	})
func (c *C) AppendAroundAction(cb CallbackAround, only ...string) {
	c.callbacks.appendAroundAction(cb, only)
}

func (c *C) AroundAction(cb CallbackAround, only ...string) {
	c.AppendAroundAction(cb, only...)
}

func (c *C) Action(name string, a ActionFunc) {
	c.actions[name] = &NamedAction{Name: name, ActionFunc: a}
}

func (c *C) Permit(params []activerecord.Attribute, names ...string) {
	for _, name := range names {
		p := c.params[name]
		c.params[name] = append(p, params...)
	}
}

func (c *C) Create(a ActionFunc) {
	c.Action(ActionCreate, a)
}

func (c *C) Update(a ActionFunc) {
	c.Action(ActionUpdate, a)
}

func (c *C) Show(a ActionFunc) {
	c.Action(ActionShow, a)
}

func (c *C) Index(a ActionFunc) {
	c.Action(ActionIndex, a)
}

func (c *C) Destroy(a ActionFunc) {
	c.Action(ActionDestroy, a)
}

type ActionController struct {
	actions actionsMap
}

func New(init func(*C)) *ActionController {
	c, err := Initialize(init)
	if err != nil {
		panic(err)
	}
	return c
}

func Initialize(init func(*C)) (*ActionController, error) {
	c := C{actions: make(actionsMap), params: make(map[string][]activerecord.Attribute)}
	init(&c)

	for actionName, action := range c.actions {
		action.Request = c.params[actionName]
		action = c.callbacks.override(action)
		c.actions[actionName] = action
	}

	return &ActionController{
		actions: c.actions.copy(),
	}, nil
}

func (c *ActionController) HasAction(actionName string) bool {
	_, ok := c.actions[actionName]
	return ok
}

func (c *ActionController) ActionMethods() []Action {
	actions := make([]Action, 0, len(c.actions))
	for _, action := range c.actions {
		actions = append(actions, action)
	}
	return actions
}

func (c *ActionController) Action(actionName string) Action {
	action, ok := c.actions[actionName]
	if !ok {
		return nil
	}
	return action
}
