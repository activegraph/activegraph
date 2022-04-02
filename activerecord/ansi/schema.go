package ansi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/activegraph/activegraph/activerecord"
)

type SchemaStatements struct {
	Conn activerecord.Conn
}

func (s *SchemaStatements) ColumnType(typeName string) (activerecord.Type, error) {
	switch strings.ToLower(typeName) {
	case "integer":
		return new(activerecord.Int64), nil
	case "varchar", "text":
		return new(activerecord.String), nil
	case "float":
		return new(activerecord.Float64), nil
	case "boolean":
		return new(activerecord.Boolean), nil
	case "datetime":
		return new(activerecord.DateTime), nil
	case "date":
		return new(activerecord.Date), nil
	case "time":
		return new(activerecord.Time), nil
	default:
		return nil, activerecord.ErrUnsupportedType{TypeName: typeName}
	}
}

func (s *SchemaStatements) ColumnDefinitions(ctx context.Context, tableName string) (
	[]activerecord.ColumnDefinition, error,
) {
	return nil, errors.New("ansi: not supported")
}

func (s *SchemaStatements) CreateTable(ctx context.Context, table *activerecord.Table) error {
	var buf strings.Builder
	fmt.Fprintf(&buf, `CREATE TABLE %q (`, table.Name())

	var primaryKey string

	for _, column := range table.Columns() {
		nativeType := column.Type.NativeType()

		if column.NotNull {
			nativeType += " NOT NULL"
		}
		if column.IsPrimaryKey {
			primaryKey = column.Name
		}

		fmt.Fprintf(&buf, `%s %s, `, column.Name, nativeType)
	}

	for _, target := range table.ForeignKeys() {
		fk := fmt.Sprintf("%s_id", strings.TrimSuffix(target, "s"))
		fmt.Fprintf(&buf, `FOREIGN KEY (%q) REFERENCES "%s" ("id"), `, fk, target)
	}

	fmt.Fprintf(&buf, `PRIMARY KEY ("%s"))`, primaryKey)
	return s.Conn.Exec(ctx, buf.String())
}

func (s *SchemaStatements) AddForeignKey(ctx context.Context, owner, target string) error {
	var buf strings.Builder

	fk := fmt.Sprintf("%s_id", strings.TrimSuffix(target, "s"))
	fmt.Fprintf(&buf, `ALTER TABLE %q ADD CONSTRAINT fk_%s_on_%s `, owner, owner, target)

	// TODO: id is not necessary a primary key.
	fmt.Fprintf(&buf, `FOREIGN KEY (%q) REFERENCES %q ("id")"`, fk, target)
	return s.Conn.Exec(ctx, buf.String())
}
