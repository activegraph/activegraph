package activerecord

import (
	"github.com/activegraph/activegraph/activesupport"
)

type Array []*ActiveRecord

func (arr Array) ToHashArray() []activesupport.Hash {
	ha := make([]activesupport.Hash, 0, len(arr))
	for _, e := range arr {
		ha = append(ha, e.ToHash())
	}
	return ha
}
