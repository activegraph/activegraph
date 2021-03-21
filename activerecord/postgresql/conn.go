package postgresql

import (
	"context"
)

type Conn struct {
}

func (c *Conn) BeginTransaction(ctx context.Context) error {
	return nil
}

func (c *Conn) CommitTransaction(ctx context.Context) error {
	return nil
}

func (c *Conn) RollbackTransaction(ctx context.Context) error {
	return nil
}

func (c *Conn) ExecInsert(ctx context.Context, sql string, args ...interface{}) error {
	return nil
}

func (c *Conn) ExecDelete(ctx context.Context, sql string, args ...interface{}) error {
	return nil
}

func (c *Conn) ExecQuery(ctx context.Context, sql string, args ...interface{}) error {
	return nil
}
