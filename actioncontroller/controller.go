package actioncontroller

import (
	"github.com/activegraph/activegraph/actiondispatch"
)

type Action interface {
	Process(*actiondispatch.Request, *actiondispatch.Response) error
}

type ActionFunc func(*actiondispatch.Request, *actiondispatch.Response) error

func (fn ActionFunc) Process(r *actiondispatch.Request, resp *actiondispatch.Response) error {
	return fn(r, resp)
}

type C struct {
	create  Action
	update  Action
	show    Action
	index   Action
	destroy Action
}

func (c *C) BeforeAction(only ...string) {
}

func (c *C) AfterAction() {
}

func (c *C) AroundAction() {
}

func (c *C) Create(a ActionFunc) {
	c.create = a
}

func (c *C) Update(a ActionFunc) {
	c.update = a
}

func (c *C) Show(a ActionFunc) {
	c.show = a
}

func (c *C) Index(a ActionFunc) {
	c.index = a
}

func (c *C) Destroy(a ActionFunc) {
	c.destroy = a
}

type ActionController struct {
	actions map[string]Action
}

func New(init func(*C)) *ActionController {
	c, err := Initialize(init)
	if err != nil {
		panic(err)
	}
	return c
}

func Initialize(init func(*C)) (*ActionController, error) {
	var c C
	init(&c)

	return &ActionController{}, nil
}

func (c *ActionController) Process(actionName string) {
}

func (c *ActionController) HasAction(actionName string) bool {
	_, ok := c.actions[actionName]
	return ok
}

func (c *ActionController) ActionMethods() []Action {
	return nil
}
