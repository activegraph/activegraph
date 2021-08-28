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
	AttributesForInspect(attrNames ...string) []activerecord.Attribute
	ReflectOnAllAssociations() []*activerecord.AssociationReflection
}

type AbstractController interface {
	ActionMethods() []Action
}

type Matcher interface {
	Matches(*Request) bool
}

type Constraints struct {
	Request  *StrongParameters
	Response *StrongParameters
	Match    Matcher
}

type Mapper interface {
	Resources(AbstractModel, AbstractController)

	// Match matches path to an action.
	Match(via, path string, action Action, constraints ...Constraints)

	Map() (http.Handler, error)
}
