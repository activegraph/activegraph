package activegraph

import (
	"github.com/activegraph/activegraph/activerecord"
)

type ActiveModel interface {
	activerecord.Persistence
	activerecord.Ownership
}
