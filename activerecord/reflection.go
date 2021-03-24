package activerecord

import (
	"github.com/pkg/errors"
)

var (
	globalReflection = NewReflection()
)

type Reflection struct {
	models map[string]*ModelSchema
}

func NewReflection() *Reflection {
	return &Reflection{models: make(map[string]*ModelSchema)}
}

func (r *Reflection) AddReflection(name string, model *ModelSchema) {
	r.models[name] = model
}

func (r *Reflection) Reflection(name string) (*ModelSchema, error) {
	model, ok := r.models[name]
	if !ok {
		return nil, errors.Errorf("unknown reflection %q", name)
	}
	return model, nil
}
