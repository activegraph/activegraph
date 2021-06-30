package actioncontroller

import (
	"github.com/activegraph/activegraph/activerecord"
)

type actionsMap map[string]*NamedAction

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

func (c *C) BeforeAction(action AnonymousAction, only ...string) {
}

func (c *C) AfterAction(action AnonymousAction, only ...string) {
}

func (c *C) AroundAction(action AnonymousAction, only ...string) {
}

func (c *C) Action(name string, a AnonymousAction) {
	c.actions[name] = &NamedAction{Name: name, AnonymousAction: a}
}

func (c *C) Permit(params []activerecord.Attribute, names ...string) {
	for _, name := range names {
		p := c.params[name]
		c.params[name] = append(p, params...)
	}
}

func (c *C) Create(a AnonymousAction) {
	c.Action(ActionCreate, a)
}

func (c *C) Update(a AnonymousAction) {
	c.Action(ActionUpdate, a)
}

func (c *C) Show(a AnonymousAction) {
	c.Action(ActionShow, a)
}

func (c *C) Index(a AnonymousAction) {
	c.Action(ActionIndex, a)
}

func (c *C) Destroy(a AnonymousAction) {
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
