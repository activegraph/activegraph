package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"

	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activesupport"
)

func init() {
	activerecord.RegisterConnectionAdapter("sqlite3", Connect)
}

type Querier interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}

type Conn struct {
	querier Querier

	db *sql.DB
	tx *sql.Tx
}

func Connect(conf activerecord.DatabaseConfig) (activerecord.Conn, error) {
	db, err := sql.Open("sqlite3", conf.Database)
	if err != nil {
		return nil, err
	}
	return &Conn{db: db, querier: db}, nil
}

func (c *Conn) Close() error {
	if c.tx != nil {
		return c.tx.Commit()
	}
	return c.db.Close()
}

func (c *Conn) BeginTransaction(ctx context.Context) (activerecord.Conn, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	fmt.Println("BEGIN TRANSACTION")
	return &Conn{db: c.db, querier: tx, tx: tx}, nil
}

func (c *Conn) CommitTransaction(ctx context.Context) error {
	if c.tx == nil {
		return errors.Errorf("no transaction is open")
	}
	fmt.Println("COMMIT TRANSACTION")
	return c.tx.Commit()
}

func (c *Conn) RollbackTransaction(ctx context.Context) error {
	if c.tx == nil {
		return errors.Errorf("no transaction is open")
	}
	fmt.Println("ROLLBACK TRANSACTION")
	return c.tx.Rollback()
}

func (c *Conn) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := c.querier.ExecContext(ctx, sql, args...)
	return err
}

func (c *Conn) ExecDelete(ctx context.Context, op *activerecord.DeleteOperation) error {
	const stmt = `DELETE FROM "%s" WHERE "%s" = '%v'`
	sql := fmt.Sprintf(stmt, op.TableName, op.PrimaryKey, op.Value)

	_, err := c.querier.ExecContext(ctx, sql)
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
	result, err := c.querier.ExecContext(ctx, c.buildInsertStmt(op))
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

func (c *Conn) ExecQuery(
	ctx context.Context, op *activerecord.QueryOperation, cb func(activesupport.Hash) bool,
) (
	err error,
) {
	fmt.Println(op.Text, op.Args)
	rws, err := c.querier.QueryContext(ctx, op.Text, op.Args...)
	if err != nil {
		return err
	}

	defer rws.Close()

	for rws.Next() {
		var (
			// Iterate over rows and scan one-by one.
			row = make(activesupport.Hash)
			// Initalize a list of interface pointer, so the Scan operation could
			// assign the results to the each element of the list.
			vals = make([]interface{}, len(op.Columns))
		)

		for i := range vals {
			vals[i] = new(interface{})
		}
		if err = rws.Scan(vals...); err != nil {
			return err
		}
		for i := range vals {
			row[op.Columns[i]] = *(vals[i]).(*interface{})
		}

		// Terminate the querying and close the reading cursor.
		if !cb(row) {
			break
		}
	}

	return nil
}
