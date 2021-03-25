package activerecord

import (
	"github.com/pkg/errors"
)

var (
	globalReflection = NewReflection()
)

type Reflection struct {
	rels map[string]*Relation
}

func NewReflection() *Reflection {
	return &Reflection{rels: make(map[string]*Relation)}
}

func (r *Reflection) AddReflection(name string, rel *Relation) {
	r.rels[name] = rel
}

func (r *Reflection) Reflection(name string) (*Relation, error) {
	rel, ok := r.rels[name]
	if !ok {
		return nil, errors.Errorf("unknown reflection %q", name)
	}
	return rel, nil
}
