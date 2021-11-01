package graphql

import (
	"github.com/vektah/gqlparser/v2/ast"
)

var Int = &ast.Definition{
	Kind: ast.Scalar,
	Name: "Int",
}

var String = &ast.Definition{
	Kind: ast.Scalar,
	Name: "String",
}

var DateTime = &ast.Definition{
	Kind: ast.Scalar,
	Name: "DateTime",
}

var List = &ast.Definition{
	Kind: "LIST",
	Name: "List",
}
