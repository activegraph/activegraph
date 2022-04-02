package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mattn/go-sqlite3"

	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activerecord/ansi"
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

	ansi.SchemaStatements

	db *sql.DB
	tx *sql.Tx
}

func Connect(conf activerecord.DatabaseConfig) (activerecord.Conn, error) {
	db, err := sql.Open("sqlite3", conf.Database)
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		db:      db,
		querier: db,
	}

	// Enable foreign keys support.
	err = conn.Exec(context.Background(), "PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}

	conn.SchemaStatements = ansi.SchemaStatements{conn}
	return conn, nil
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

	conn := &Conn{db: c.db, querier: tx, tx: tx}
	conn.SchemaStatements = ansi.SchemaStatements{conn}

	return conn, nil
}

func (c *Conn) CommitTransaction(ctx context.Context) error {
	if c.tx == nil {
		return fmt.Errorf("no transaction is open")
	}
	fmt.Println("COMMIT TRANSACTION")
	return c.tx.Commit()
}

func (c *Conn) RollbackTransaction(ctx context.Context) error {
	if c.tx == nil {
		return fmt.Errorf("no transaction is open")
	}
	fmt.Println("ROLLBACK TRANSACTION")
	return c.tx.Rollback()
}

func (c *Conn) Exec(ctx context.Context, sql string, args ...interface{}) error {
	fmt.Println(">>>", sql, args)
	_, err := c.querier.ExecContext(ctx, sql, args...)
	return err
}

func (c *Conn) ExecDelete(ctx context.Context, op *activerecord.DeleteOperation) error {
	const stmt = `DELETE FROM "%s" WHERE "%s" = '%v'`
	sql := fmt.Sprintf(stmt, op.TableName, op.PrimaryKey, op.Value)

	fmt.Println(sql)
	_, err := c.querier.ExecContext(ctx, sql)
	return err
}

func (c *Conn) buildInsertStmt(op *activerecord.InsertOperation) (string, error) {
	var (
		colBuf strings.Builder
		valBuf strings.Builder
	)

	colPos, colNum := 0, len(op.ColumnValues)
	for _, col := range op.ColumnValues {
		val, err := col.Type.Serialize(col.Value)
		if err != nil {
			return "", err
		}

		colfmt, valfmt := `"%s", `, `'%v', `
		if colPos == colNum-1 {
			colfmt, valfmt = `"%s"`, `'%v'`
		}

		fmt.Fprintf(&colBuf, colfmt, col.Name)
		fmt.Fprintf(&valBuf, valfmt, val)
		colPos++
	}

	const stmt = `INSERT INTO "%s" (%s) VALUES (%s)`
	return fmt.Sprintf(stmt, op.TableName, colBuf.String(), valBuf.String()), nil
}

func (c *Conn) buildUpdateStmt(op *activerecord.UpdateOperation) (string, error) {
	var (
		stmtBuf strings.Builder
		pk      interface{}
	)

	colNum := len(op.ColumnValues)
	for colPos, col := range op.ColumnValues {
		val, err := col.Type.Serialize(col.Value)
		if err != nil {
			return "", err
		}

		valfmt := `"%s" = '%v', `
		if colPos == colNum-1 {
			valfmt = `"%s" = '%v'`
		}
		if col.Name == op.PrimaryKey {
			pk = val
		}

		fmt.Fprintf(&stmtBuf, valfmt, col.Name, val)
	}

	const stmt = `UPDATE "%s" SET %s WHERE "%s" = '%v'`
	return fmt.Sprintf(stmt, op.TableName, stmtBuf.String(), op.PrimaryKey, pk), nil
}

func (c *Conn) ExecInsert(ctx context.Context, op *activerecord.InsertOperation) (
	id interface{}, err error,
) {
	stmt, err := c.buildInsertStmt(op)
	if err != nil {
		return 0, err
	}
	fmt.Println(stmt)

	result, err := c.querier.ExecContext(ctx, stmt)
	if err != nil {
		switch err := err.(type) {
		case sqlite3.Error:
			switch err.ExtendedCode {
			case sqlite3.ErrConstraintPrimaryKey, sqlite3.ErrConstraintUnique:
				return 0, &activerecord.ErrRecordNotUnique{Err: err}
			default:
				return 0, err
			}
		default:
			return 0, err
		}
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows != 1 {
		return 0, fmt.Errorf("expected single row affected, got %d rows affected", rows)
	}

	return result.LastInsertId()
}

func (c *Conn) ExecUpdate(ctx context.Context, op *activerecord.UpdateOperation) error {
	stmt, err := c.buildUpdateStmt(op)
	if err != nil {
		return err
	}
	fmt.Println(stmt)

	result, err := c.querier.ExecContext(ctx, stmt)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("expected single row affected, got %d rows affected", rows)
	}
	return nil
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
			// Initalize a list of interfaces, so the Scan operation could
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

func (c *Conn) ColumnDefinitions(ctx context.Context, tableName string) (
	[]activerecord.ColumnDefinition, error,
) {
	stmt := fmt.Sprintf("PRAGMA table_info('%s')", tableName)
	rws, err := c.querier.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}

	defer rws.Close()

	var definitions []activerecord.ColumnDefinition
	for rws.Next() {
		var (
			cid, notnull, pk int
			fname, ftype     string
			defaultValue     interface{}
		)

		err := rws.Scan(&cid, &fname, &ftype, &notnull, &defaultValue, &pk)
		if err != nil {
			return nil, err
		}

		columnType, err := c.ColumnType(ftype)
		if err != nil {
			return nil, err
		}

		definitions = append(definitions, activerecord.ColumnDefinition{
			Name:         fname,
			Type:         columnType,
			NotNull:      notnull == 1,
			IsPrimaryKey: pk == 1,
		})
	}
	if len(definitions) == 0 {
		return nil, activerecord.ErrTableNotExist{TableName: tableName}
	}
	return definitions, nil
}

func (c *Conn) AddForeignKey(ctx context.Context, owner, target string) error {
	// SQLite does not support adding a foreign key constraint, which
	// is implemented in ANSI schema statements, therefore we need to
	// drop table and create a new one with foreign keys inside.
	stmts := []string{
		`PRAGMA foreign_keys = OFF`,
		fmt.Sprintf(`ALTER TABLE %q RENAME TO "temp_%s"`, owner, owner),
	}

	columns, err := c.ColumnDefinitions(ctx, owner)
	if err != nil {
		return err
	}

	for _, stmt := range stmts {
		if err := c.Exec(ctx, stmt); err != nil {
			return err
		}
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, `CREATE TABLE %q (`, owner)

	var primaryKey string

	for _, column := range columns {
		nativeType := column.Type.NativeType()
		if column.NotNull {
			nativeType += " NOT NULL"
		}
		if column.IsPrimaryKey {
			primaryKey = column.Name
		}
		fmt.Fprintf(&buf, `%s %s, `, column.Name, nativeType)
	}

	// TODO: Add all foreign keys as well.
	fk := fmt.Sprintf("%s_id", strings.TrimSuffix(target, "s"))

	fmt.Fprintf(&buf, `FOREIGN KEY (%q) REFERENCES "%s" ("id"), `, fk, target)
	fmt.Fprintf(&buf, `PRIMARY KEY (%q))`, primaryKey)

	stmts = []string{
		buf.String(), // CREATE TABLE ...
		fmt.Sprintf(`INSERT INTO %q SELECT * FROM "temp_%s"`, owner, owner),
		fmt.Sprintf(`DROP TABLE "temp_%s"`, owner),
		fmt.Sprintf(`PRAGMA foreign_keys = ON`),
	}

	for _, stmt := range stmts {
		if err := c.Exec(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}
