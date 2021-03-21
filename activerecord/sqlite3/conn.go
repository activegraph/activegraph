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

func Open() *Conn {
	db, err := sql.Open("sqlite3", ":memory:")
	// TODO: remove this.
	db.ExecContext(context.TODO(), `CREATE TABLE books (pages integer, title varchar);`)

	if err != nil {
		panic(err)
	}
	return &Conn{db: db}
}

func (c *Conn) BuildInsertStmt(op *activerecord.InsertOperation) string {
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

	const stmt = `INSERT INTO "%s" (%s) VALUES (%s);`
	return fmt.Sprintf(stmt, op.TableName, colBuf.String(), valBuf.String())
}

func (c *Conn) ExecInsert(ctx context.Context, sql string, args ...interface{}) error {
	fmt.Println(sql)
	result, err := c.db.ExecContext(ctx, sql)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return errors.Errorf("expected single row affected, got %d rows affected", rows)
	}
	return nil
}
