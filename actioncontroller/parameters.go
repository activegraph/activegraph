package actioncontroller

import (
	"github.com/activegraph/activegraph/activerecord"
)

type Parameters map[string]interface{}

func (p Parameters) Get(key string) Parameters {
	val, ok := p[key]
	if !ok {
		return nil
	}
	params, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}
	return params
}

func (p Parameters) ToH() map[string]interface{} {
	return (map[string]interface{})(p)
}

type StrongParameters struct {
	Attributes []activerecord.Attribute
}

func (p *StrongParameters) Permit(names ...string) *StrongParameters {
	attrs := make([]activerecord.Attribute, 0, len(names))
	for _, attr := range p.Attributes {
		attrs = append(attrs, attr)
	}
	return &StrongParameters{Attributes: attrs}
}

func Require(rel *activerecord.Relation) *StrongParameters {
	return &StrongParameters{Attributes: rel.AttributesForInspect()}
}
