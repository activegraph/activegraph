package activerecord

import (
	"context"
	"fmt"

	"github.com/activegraph/activegraph/activesupport"
)

type ErrTableNotExist struct {
	TableName string
}

func (e ErrTableNotExist) Error() string {
	return fmt.Sprintf("ErrTableNotExist: '%s'", e.TableName)
}

type Persistence interface {
	Insert() (*ActiveRecord, error)
	Update() (*ActiveRecord, error)
	Delete() error
	IsPersisted() bool
}

type InsertOperation struct {
	TableName      string
	ColumnValues   []ColumnValue
	OnDuplicate    string
	ConflictTarget string
}

type UpdateOperation struct {
	TableName    string
	PrimaryKey   string
	ColumnValues []ColumnValue
}

type DeleteOperation struct {
	TableName  string
	PrimaryKey string
	Value      interface{}
}

type Dependency struct {
	TableName  string
	ForeignKey string
	PrimaryKey string
}

type QueryOperation struct {
	Text    string
	Args    []interface{}
	Columns []string
}

type ColumnValue struct {
	Name  string
	Type  Type
	Value interface{}
}

type ColumnDefinition struct {
	Name         string
	Type         Type
	NotNull      bool
	IsPrimaryKey bool
}

type TransactionStatements interface {
	BeginTransaction(ctx context.Context) (Conn, error)
	CommitTransaction(ctx context.Context) error
	RollbackTransaction(ctx context.Context) error
}

type DatabaseStatements interface {
	ExecInsert(ctx context.Context, op *InsertOperation) (id interface{}, err error)
	ExecUpdate(ctx context.Context, op *UpdateOperation) (err error)
	ExecDelete(ctx context.Context, op *DeleteOperation) (err error)
	ExecQuery(ctx context.Context, op *QueryOperation, cb func(activesupport.Hash) bool) (err error)
}

type SchemaStatements interface {
	CreateTable(ctx context.Context, table *Table) error
	AddForeignKey(ctx context.Context, owner, target string) error

	ColumnType(typeName string) (Type, error)
	ColumnDefinitions(ctx context.Context, tableName string) ([]ColumnDefinition, error)
}

type Conn interface {
	TransactionStatements
	DatabaseStatements
	SchemaStatements

	Close() error
}

type errConn struct {
	err error
}

// TransactionStatements
func (c *errConn) BeginTransaction(ctx context.Context) (Conn, error) {
	return nil, c.err
}

func (c *errConn) CommitTransaction(ctx context.Context) error {
	return c.err
}

func (c *errConn) RollbackTransaction(ctx context.Context) error {
	return c.err
}

// DatabaseStatements
func (c *errConn) ExecInsert(context.Context, *InsertOperation) (interface{}, error) {
	return nil, c.err
}

func (c *errConn) ExecUpdate(context.Context, *UpdateOperation) error {
	return c.err
}

func (c *errConn) ExecDelete(context.Context, *DeleteOperation) error {
	return c.err
}

func (c *errConn) ExecQuery(context.Context, *QueryOperation, func(activesupport.Hash) bool) error {
	return c.err
}

// SchemaStatements
func (c *errConn) ColumnType(typeName string) (Type, error) {
	return nil, c.err
}

func (c *errConn) ColumnDefinitions(ctx context.Context, tableName string) ([]ColumnDefinition, error) {
	return nil, c.err
}

func (c *errConn) CreateTable(ctx context.Context, table *Table) error {
	return c.err
}

func (c *errConn) AddForeignKey(ctx context.Context, owner, target string) error {
	return c.err
}

func (c *errConn) Close() error {
	return c.err
}
