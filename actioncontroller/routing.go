package actioncontroller

import (
	"net/http"

	"github.com/activegraph/activegraph/activerecord"
)

type AbstractModel interface {
	Name() string
	PrimaryKey() string
	AttributeNames() []string
	AttributeForInspect(attrName string) activerecord.Attribute
}

type AbstractController interface {
	ActionMethods() []Action
}

type Mapper interface {
	Resources(AbstractModel, AbstractController)
	Map() (http.Handler, error)
}
