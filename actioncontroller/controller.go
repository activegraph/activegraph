package actioncontroller

import (
	"github.com/activegraph/activegraph/activerecord"
)

type Create struct {
}

type CreateAction func(*Create) (*activerecord.ActiveRecord, error)

type Update struct {
}

type UpdateAction func(*Update) (*activerecord.ActiveRecord, error)

type C struct {
	create CreateAction
	update UpdateAction
}

func (c *C) BeforeAction(only ...string) {
}

func (c *C) AfterAction() {
}

func (c *C) AroundAction() {
}

func (c *C) Create(a CreateAction) {
	c.create = a
}

func (c *C) Update(a UpdateAction) {
	c.update = a
}

func (c *C) Show() {
}

func (c *C) Index() {
}

func (c *C) Destroy() {
}

type ActionController struct {
}

func New(rel *activerecord.Relation, init func(*C)) *ActionController {
	c, err := Initialize(rel, init)
	if err != nil {
		panic(err)
	}
	return c
}

func Initialize(rel *activerecord.Relation, init func(*C)) (*ActionController, error) {
	c := C{}
	init(&c)

	return &ActionController{}, nil
}
