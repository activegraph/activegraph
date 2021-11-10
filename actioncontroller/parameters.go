package actioncontroller

import (
	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activesupport"
)

type Parameters activesupport.Hash

func (p Parameters) Get(key string) Parameters {
	val, ok := p[key]
	if !ok {
		return nil
	}
	params, ok := val.(activesupport.Hash)
	if !ok {
		return nil
	}
	return Parameters(params)
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
