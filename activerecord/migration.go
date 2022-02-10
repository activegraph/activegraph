package activerecord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	. "github.com/activegraph/activegraph/activesupport"
)

const (
	SchemaMigrationsName = "schema_migrations"
)

type Table struct {
	name       string
	primaryKey string

	columns map[string]Type
}

func (tb *Table) PrimaryKey(primaryKey string) {
	tb.primaryKey = primaryKey
}

func (tb *Table) DefineColumn(name string, t Type) {
	tb.columns[name] = t
}

func (tb *Table) Int64(name string) {
	tb.DefineColumn(name, new(Int64))
}

func (tb *Table) String(name string) {
	tb.DefineColumn(name, new(String))
}

func (tb *Table) DateTime(name string) {
	tb.DefineColumn(name, new(DateTime))
}

type M struct {
	tables     map[string]Table
	references map[string]string

	connections *connectionHandler
}

func (m *M) TableExists(tableName string) Result[bool] {
	return FutureOk(true).AndThen(
		func(bool) Result[bool] {
			conn, err := m.connections.RetrieveConnection(primaryConnectionName)
			if err != nil {
				return Err[bool](err)
			}

			_, err = conn.ColumnDefinitions(context.TODO(), tableName)
			if errors.Is(err, ErrTableNotExist{tableName}) {
				return Ok(false)
			}
			if err != nil {
				fmt.Println("??", err)
				return Err[bool](err)
			}
			return Ok(true)
		},
	)
}

func (m *M) CreateTable(name string, init func(*Table)) {
	table := Table{
		name:    name,
		columns: make(map[string]Type),
	}
	init(&table)

	m.tables[name] = table
}

func (m *M) AddForeignKey(owner, target string) {
	m.references[owner] = target
}

func (m *M) prepareSQL(table *Table) string {
	var buf strings.Builder

	fmt.Fprintf(&buf, `CREATE TABLE `)
	fmt.Fprintf(&buf, `%s (`, table.name)

	primaryKey := table.primaryKey
	if primaryKey == "" {
		primaryKey = "id"
		table.Int64(primaryKey)
	}

	for columnName, columnType := range table.columns {
		var sqlType string
		switch columnType.(type) {
		case *String:
			sqlType = "VARCHAR"
		case *DateTime:
			sqlType = "DATETIME"
		case *Int64:
			sqlType = "INTEGER"
		}

		if columnName == primaryKey {
			sqlType += " NOT NULL"
		}

		fmt.Fprintf(&buf, `%s %s, `, columnName, sqlType)
	}

	if targetTable, ok := m.references[table.name]; ok {
		target := strings.TrimSuffix(targetTable, "s")

		// TODO: id is not necessary a primary key.
		fmt.Fprintf(&buf, `%s_id INTEGER, `, target)
		fmt.Fprintf(&buf, `FOREIGN KEY ("%s_id") REFERENCES "%s" ("id") `, target, targetTable)
	}
	fmt.Fprintf(&buf, `PRIMARY KEY ("%s"))`, primaryKey)
	return buf.String()
}

func Migrate(id string, init func(m *M)) {
	m := M{
		tables:      make(map[string]Table),
		references:  make(map[string]string),
		connections: globalConnectionHandler,
	}

	init(&m)

	// Before applying further migrations, ensure that table exists.
	// This operation should be unwrapped within a migration transaction.
	schema := m.TableExists(SchemaMigrationsName).AndThen(
		func(exists bool) Result[bool] {
			if !exists {
				m.CreateTable(SchemaMigrationsName, func(t *Table) {
					t.PrimaryKey("version")
					t.String("version")
					t.DateTime("created_at")
				})
			}
			return Ok(true)
		},
	)

	// TODO: Use the specified connection name, instead of the default.
	// conn, err := m.connections.RetrieveConnection(primaryConnectionName)
	err := m.connections.Transaction(context.TODO(), func() error {
		if schema.IsErr() {
			return schema.Err()
		}

		conn, err := m.connections.RetrieveConnection(primaryConnectionName)
		if err != nil {
			return err
		}

		schemaTable, ok := m.tables[SchemaMigrationsName]
		if ok {
			delete(m.tables, SchemaMigrationsName)

			err := conn.Exec(context.TODO(), m.prepareSQL(&schemaTable))
			if err != nil {
				return err
			}
		}

		SchemaMigration := New("schema_migration")
		migration := SchemaMigration.Create(Hash{"version": id, "created_at": time.Now()})

		if errors.Is(migration.Err(), new(ErrRecordNotUnique)) {
			// Commit the transaction since it's already applied.
			return nil
		} else if migration.IsErr() {
			return migration.Err()
		}

		for _, table := range m.tables {
			sql := m.prepareSQL(&table)
			fmt.Println(">>>", sql)

			err = conn.Exec(context.TODO(), sql)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}
}
