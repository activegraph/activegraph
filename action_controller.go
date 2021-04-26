package activegraph

import (
	"context"
)

type Params map[string]interface{}

type ActionCreate interface {
	Create(ctx context.Context, input Params) (ActiveModel, error)
}

type ActionEdit interface {
	Edit(ctx context.Context, id interface{}, input Params) (ActiveModel, error)
}

type ActionShow interface {
	Show(ctx context.Context, id interface{}) (ActiveModel, error)
}

type ActionIndex interface {
	Index(ctx context.Context, input Params) ([]ActiveModel, error)
}

type ActionDestroy interface {
	Destroy(ctx context.Context, id interface{}) error
}

type ActionController interface {
}

func NewActionController(c ActionController) {
	if _, ok := c.(ActionCreate); ok {
		// Generate "create{ModelName}" mutation
	}
	if _, ok := c.(ActionEdit); ok {
		// Generate "update{ModelName}" mutation
	}
	if _, ok := c.(ActionDestroy); ok {
		// Generate "delete{ModelName}" mutation
	}
	if _, ok := c.(ActionShow); ok {
		// Generate "{modelName}" query
	}
	if _, ok := c.(ActionIndex); ok {
		// Generate "{modelNames}" query
	}
}
