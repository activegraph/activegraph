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
	fmt.Println(c.buildInsertStmt(op))
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

func (c *Conn) ExecQuery(ctx context.Context, op *activerecord.QueryOperation) (
	cols map[string]interface{}, err error,
) {
	const stmt = `SELECT %s FROM "%s" WHERE "%s" = '%v'`
	sql := fmt.Sprintf(stmt, strings.Join(op.Columns, ", "), op.TableName, op.PrimaryKey, op.Value)

	rows, err := c.db.QueryContext(ctx, sql)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	cols = make(map[string]interface{})

	for rows.Next() {
		vals := make([]interface{}, len(op.Columns))
		for i := range vals {
			vals[i] = new(interface{})
		}

		if err = rows.Scan(vals...); err != nil {
			return nil, err
		}

		for i := range vals {
			cols[op.Columns[i]] = *(vals[i]).(*interface{})
		}
	}

	return cols, nil
}
