package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activerecord/ansi"
	"github.com/mattn/go-sqlite3"
)

func init() {
	activerecord.RegisterConnectionAdapter("sqlite3", Connect)
}

type Conn struct {
	ansi.ConnectionStatements
	ansi.SchemaStatements
	ansi.DatabaseStatements

	db *sql.DB
	tx *sql.Tx
}

func Connect(conf activerecord.DatabaseConfig) (activerecord.Conn, error) {
	db, err := sql.Open("sqlite3", conf.Database)
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		db:                   db,
		ConnectionStatements: db,
		SchemaStatements:     ansi.SchemaStatements{db},
		DatabaseStatements:   ansi.DatabaseStatements{db},
	}

	// Enable foreign keys support.
	_, err = db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}
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

	return &Conn{
		db:                   c.db,
		tx:                   tx,
		ConnectionStatements: tx,
		SchemaStatements:     ansi.SchemaStatements{tx},
		DatabaseStatements:   ansi.DatabaseStatements{tx},
	}, nil
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

func (c *Conn) ExecInsert(ctx context.Context, op *activerecord.InsertOperation) (
	id interface{}, err error,
) {
	id, err = c.DatabaseStatements.ExecInsert(ctx, op)
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

	return id, err
}

func (c *Conn) ColumnDefinitions(ctx context.Context, tableName string) (
	[]activerecord.ColumnDefinition, error,
) {
	stmt := fmt.Sprintf("PRAGMA table_info('%s')", tableName)
	rws, err := c.ConnectionStatements.QueryContext(ctx, stmt)
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
		if _, err := c.ExecContext(ctx, stmt); err != nil {
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
		if _, err := c.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}
