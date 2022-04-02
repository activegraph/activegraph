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
	name        string
	primaryKey  string
	foreignKeys []string

	columns map[string]Type
}

func (tb *Table) Name() string {
	return tb.name
}

func (tb *Table) ForeignKeys() []string {
	return tb.foreignKeys
}

func (tb *Table) Columns() (columns []ColumnDefinition) {
	for columnName, columnType := range tb.columns {
		columns = append(columns, ColumnDefinition{
			Name:         columnName,
			Type:         columnType,
			IsPrimaryKey: columnName == tb.primaryKey,
		})
	}

	if _, ok := tb.columns[tb.primaryKey]; !ok {
		columns = append(columns, ColumnDefinition{
			Name:         "id",
			Type:         new(Int64),
			IsPrimaryKey: true,
			NotNull:      true,
		})
	}
	return columns
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

func (tb *Table) ForeignKey(target string) {
	tb.foreignKeys = append(tb.foreignKeys, target)
}

type References struct {
	ForeignKey bool
}

func (tb *Table) References(target string, init ...References) {
	switch len(init) {
	case 0:
	case 1:
		if init[0].ForeignKey {
			tb.ForeignKey(target)
		}
	default:
		panic(ErrMultipleVariadicArguments{Name: "init"})
	}

	ref := fmt.Sprintf("%s_id", strings.TrimSuffix(target, "s"))
	tb.DefineColumn(ref, new(Int64))
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
	if table, ok := m.tables[owner]; ok {
		// If it's a new table, add a foreign key directly into the table definition.
		table.ForeignKey(target)
		m.tables[owner] = table
	} else {
		m.references[owner] = target
	}
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

			err := conn.CreateTable(context.TODO(), &schemaTable)
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
			err = conn.CreateTable(context.TODO(), &table)
			if err != nil {
				return err
			}
		}

		for owner := range m.references {
			err = conn.AddForeignKey(context.TODO(), owner, m.references[owner])
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
