package activerecord

import (
	"fmt"
)

var (
	globalReflection = NewReflection()
)

type Reflection struct {
	rels   map[string]*Relation
	tables map[string]string
}

func NewReflection() *Reflection {
	return &Reflection{
		rels:   make(map[string]*Relation),
		tables: make(map[string]string),
	}
}

func (r *Reflection) AddReflection(name string, rel *Relation) {
	r.rels[name] = rel
	r.tables[rel.tableName] = name
}

func (r *Reflection) Reflection(name string) (*Relation, error) {
	rel, ok := r.rels[name]
	if !ok {
		return nil, fmt.Errorf("unknown relation %q", name)
	}
	return rel, nil
}
