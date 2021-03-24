package postgresql

import (
	"context"

	"github.com/activegraph/activegraph/activerecord"
)

type Conn struct {
}

func (c *Conn) ExecDelete(ctx context.Context, op *activerecord.DeleteOperation) error {
	return nil
}

func (c *Conn) ExecInsert(ctx context.Context, op *activerecord.InsertOperation) (
	id interface{}, err error,
) {
	return nil, nil
}
