package actioncontroller

import (
	"github.com/activegraph/activegraph/actiondispatch"
	"github.com/activegraph/activegraph/activerecord"
)

type actionsMap map[string]*actiondispatch.NamedAction

func (m actionsMap) copy() actionsMap {
	mm := make(actionsMap, len(m))
	for name, action := range m {
		mm[name] = action
	}
	return mm
}

type C struct {
	actions actionsMap
	params  map[string][]activerecord.Attribute
}

func (c *C) BeforeAction(only ...string) {
}

func (c *C) AfterAction() {
}

func (c *C) AroundAction() {
}

func (c *C) Action(name string, a actiondispatch.AnonymousAction) {
	c.actions[name] = &actiondispatch.NamedAction{Name: name, AnonymousAction: a}
}

func (c *C) Permit(params []activerecord.Attribute, names ...string) {
	for _, name := range names {
		p := c.params[name]
		c.params[name] = append(p, params...)
	}
}

func (c *C) Create(a actiondispatch.AnonymousAction) {
	c.Action(actiondispatch.ActionCreate, a)
}

func (c *C) Update(a actiondispatch.AnonymousAction) {
	c.Action(actiondispatch.ActionUpdate, a)
}

func (c *C) Show(a actiondispatch.AnonymousAction) {
	c.Action(actiondispatch.ActionShow, a)
}

func (c *C) Index(a actiondispatch.AnonymousAction) {
	c.Action(actiondispatch.ActionIndex, a)
}

func (c *C) Destroy(a actiondispatch.AnonymousAction) {
	c.Action(actiondispatch.ActionDestroy, a)
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
	}

	return &ActionController{
		actions: c.actions.copy(),
	}, nil
}

func (c *ActionController) HasAction(actionName string) bool {
	_, ok := c.actions[actionName]
	return ok
}

func (c *ActionController) ActionMethods() []actiondispatch.Action {
	actions := make([]actiondispatch.Action, 0, len(c.actions))
	for _, action := range c.actions {
		actions = append(actions, action)
	}
	return actions
}
