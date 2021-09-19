package graphql

import (
	"github.com/activegraph/activegraph/activerecord"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

var DateTime = graphql.NewScalar(graphql.ScalarConfig{
	Name: "DateTime",
	Description: "The `DateTime` scalar type represents a DateTime." +
		" The DateTime is serialized as an RFC 3339 quoted string",
	Serialize: func(value interface{}) interface{} {
		value, err := new(activerecord.DateTime).Serialize(value)
		if err != nil {
			panic(err)
		}
		return value
	},
	ParseValue: func(value interface{}) interface{} {
		value, err := new(activerecord.DateTime).Deserialize(value)
		if err != nil {
			panic(err)
		}
		return value
	},
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch valueAST := valueAST.(type) {
		case *ast.StringValue:
			value, err := new(activerecord.DateTime).Deserialize(valueAST.Value)
			// TODO: this is unacceptable implementation and must be reworked.
			if err != nil {
				panic(err)
			}
			return value
		}
		return nil
	},
})
