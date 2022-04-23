package ansi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/activegraph/activegraph/activerecord"
	. "github.com/activegraph/activegraph/activesupport"
)

type ConnectionStatements interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}

type DatabaseStatements struct {
	Conn ConnectionStatements
}

func (s *DatabaseStatements) buildInsertStmt(op *activerecord.InsertOperation) (string, error) {
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

func (s *DatabaseStatements) ExecInsert(ctx context.Context, op *activerecord.InsertOperation) (
	id interface{}, err error,
) {
	stmt, err := s.buildInsertStmt(op)
	if err != nil {
		return 0, err
	}
	fmt.Println(stmt)

	result, err := s.Conn.ExecContext(ctx, stmt)
	if err != nil {
		return 0, err
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

func (s *DatabaseStatements) buildUpdateStmt(op *activerecord.UpdateOperation) (string, error) {
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

func (s *DatabaseStatements) ExecUpdate(
	ctx context.Context, op *activerecord.UpdateOperation,
) error {
	stmt, err := s.buildUpdateStmt(op)
	if err != nil {
		return err
	}
	fmt.Println(stmt)

	result, err := s.Conn.ExecContext(ctx, stmt)
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

func (s *DatabaseStatements) ExecDelete(ctx context.Context, op *activerecord.DeleteOperation) error {
	const stmt = `DELETE FROM "%s" WHERE "%s" = '%v'`
	sql := fmt.Sprintf(stmt, op.TableName, op.PrimaryKey, op.Value)
	_, err := s.Conn.ExecContext(ctx, sql)
	return err
}

func (s *DatabaseStatements) ExecQuery(
	ctx context.Context, op *activerecord.QueryOperation, cb func(Hash) bool,
) (
	err error,
) {
	fmt.Println(op.Text, op.Args)
	rws, err := s.Conn.QueryContext(ctx, op.Text, op.Args...)
	if err != nil {
		return err
	}

	defer rws.Close()

	for rws.Next() {
		var (
			// Iterate over rows and scan one-by one.
			row = make(Hash)
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

type SchemaStatements struct {
	Conn ConnectionStatements
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
	_, err := s.Conn.ExecContext(ctx, buf.String())
	return err
}

func (s *SchemaStatements) AddForeignKey(ctx context.Context, owner, target string) error {
	var buf strings.Builder

	fk := fmt.Sprintf("%s_id", strings.TrimSuffix(target, "s"))
	fmt.Fprintf(&buf, `ALTER TABLE %q ADD CONSTRAINT fk_%s_on_%s `, owner, owner, target)

	// TODO: id is not necessary a primary key.
	fmt.Fprintf(&buf, `FOREIGN KEY (%q) REFERENCES %q ("id")"`, fk, target)
	_, err := s.Conn.ExecContext(ctx, buf.String())
	return err
}
