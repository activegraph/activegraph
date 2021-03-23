package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"

	"github.com/activegraph/activegraph/activerecord"
)

type Conn struct {
	db *sql.DB
}

func Open(dataSourceName string) (*Conn, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Conn{db: db}, nil
}

func (c *Conn) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := c.db.ExecContext(ctx, sql, args...)
	return err
}

func (c *Conn) ExecDelete(ctx context.Context, op *activerecord.DeleteOperation) error {
	const stmt = `DELETE FROM "%s" WHERE "%s" = '%v'`
	sql := fmt.Sprintf(stmt, op.TableName, op.PrimaryKey, op.Value)

	_, err := c.db.ExecContext(ctx, sql)
	return err
}

func (c *Conn) buildInsertStmt(op *activerecord.InsertOperation) string {
	var (
		colBuf strings.Builder
		valBuf strings.Builder
	)

	colPos, colNum := 0, len(op.Values)
	for col, val := range op.Values {
		colfmt, valfmt := `"%s", `, `'%v', `
		if colPos == colNum-1 {
			colfmt, valfmt = `"%s"`, `'%v'`
		}
		fmt.Fprintf(&colBuf, colfmt, col)
		fmt.Fprintf(&valBuf, valfmt, val)
		colPos++
	}

	const stmt = `INSERT INTO "%s" (%s) VALUES (%s)`
	return fmt.Sprintf(stmt, op.TableName, colBuf.String(), valBuf.String())
}

func (c *Conn) ExecInsert(ctx context.Context, op *activerecord.InsertOperation) (
	id interface{}, err error,
) {
	result, err := c.db.ExecContext(ctx, c.buildInsertStmt(op))
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows != 1 {
		return 0, errors.Errorf("expected single row affected, got %d rows affected", rows)
	}

	return result.LastInsertId()
}
