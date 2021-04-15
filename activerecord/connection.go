package activerecord

import (
	"context"
	"fmt"
	"sync"

	"github.com/activegraph/activegraph/internal"
	"github.com/pkg/errors"
)

const (
	primaryConnectionName = "primary"
)

var (
	globalConnectionHandler *connectionHandler
)

func init() {
	globalConnectionHandler = newConnectionHandler()
}

// ErrAdapterNotFound is returned when Active Record cannot find database
// specified database adapter.
type ErrAdapterNotFound struct {
	Adapter string
}

func (e *ErrAdapterNotFound) Error() string {
	return fmt.Sprintf("adapter %q not found", e.Adapter)
}

// ErrConnectionNotEstablished is returned when connection to the database
// is not established.
type ErrConnectionNotEstablished struct {
	Name string
}

func (e *ErrConnectionNotEstablished) Error() string {
	return fmt.Sprintf("connection %q has not been established", e.Name)
}

type DatabaseConfig struct {
	Name     string
	Adapter  string
	Host     string
	Username string
	Password string
	Database string
}

type ConnectionAdapter func(DatabaseConfig) (Conn, error)

// connectionHandler is responsible of keeping the state of established connections
// adapters registration routine.
type connectionHandler struct {
	adapters map[string]ConnectionAdapter
	conns    map[string]Conn
	tx       map[uint64]Conn
	mu       sync.RWMutex
}

func newConnectionHandler() *connectionHandler {
	return &connectionHandler{
		adapters: make(map[string]ConnectionAdapter),
		conns:    make(map[string]Conn),
		tx:       make(map[uint64]Conn),
	}
}

func (h *connectionHandler) RegisterConnectionAdapter(
	adapter string, ca ConnectionAdapter,
) error {
	if _, dup := h.adapters[adapter]; dup {
		return fmt.Errorf("duplicate connection adapter")
	}
	h.adapters[adapter] = ca
	return nil
}

func (h *connectionHandler) ConnectionSpecificationName(name string) string {
	return fmt.Sprintf("%s/%d", name, internal.GoroutineID())
}

func (h *connectionHandler) Transaction(ctx context.Context, fn func() error) error {
	conn, err := h.RetrieveConnection(primaryConnectionName)
	if err != nil {
		return err
	}

	conn, err = conn.BeginTransaction(ctx)
	if err != nil {
		return err
	}

	// Return the connection back to the pool. From that moment, all database
	// operations for this connection will be finished with an error.
	defer conn.Close()

	goroutineID := internal.GoroutineID()
	h.mu.Lock()
	h.tx[goroutineID] = conn
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.tx, goroutineID)
	}()

	if err = fn(); err != nil {
		if e := conn.RollbackTransaction(ctx); e != nil {
			err = errors.WithMessage(err, e.Error())
		}
		return err
	}

	return conn.CommitTransaction(ctx)
}

func (h *connectionHandler) EstablishConnection(c DatabaseConfig) (Conn, error) {
	newConnection, ok := h.adapters[c.Adapter]
	if !ok {
		return nil, &ErrAdapterNotFound{Adapter: c.Adapter}
	}

	conn, err := newConnection(c)
	if err != nil {
		return nil, err
	}

	if c.Name == "" {
		c.Name = primaryConnectionName
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, dup := h.conns[c.Name]; dup {
		return nil, fmt.Errorf("connection %q already established", c.Name)
	}

	h.conns[c.Name] = conn
	return conn, nil
}

func (h *connectionHandler) RetrieveConnection(name string) (Conn, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	tx, ok := h.tx[internal.GoroutineID()]
	if ok {
		return tx, nil
	}

	conn, ok := h.conns[name]
	if !ok {
		return nil, &ErrConnectionNotEstablished{Name: name}
	}
	return conn, nil
}

func (h *connectionHandler) RemoveConnection(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	conn, ok := h.conns[name]
	if !ok {
		return &ErrConnectionNotEstablished{Name: name}
	}

	delete(h.conns, name)
	return conn.Close()
}

func RegisterConnectionAdapter(adapter string, ca ConnectionAdapter) {
	err := globalConnectionHandler.RegisterConnectionAdapter(adapter, ca)
	if err != nil {
		panic(err)
	}
}

// EstablishConnection establishes connection to the database. Accepts a configuration
// as input where Adapter key must be specified with the name of a database adapter
// (in lower-case).
//
// Example for PostgreSQL database:
//
//	activerecord.EstablishConnection(activerecord.DatabaseConfig{
//		Adapter:  "postgresql",
//		Host:     "localhost",
//		Username: "pguser",
//		Password: "pgpass",
//		Database: "somedatabase",
//	})
func EstablishConnection(c DatabaseConfig) (Conn, error) {
	return globalConnectionHandler.EstablishConnection(c)
}

func RetrieveConnection(name string) (Conn, error) {
	return globalConnectionHandler.RetrieveConnection(name)
}

func RemoveConnection(name string) error {
	return globalConnectionHandler.RemoveConnection(name)
}

// Transaction runs the given block in a database transaction, and returns the
// result of the function.
func Transaction(ctx context.Context, fn func() error) error {
	return globalConnectionHandler.Transaction(ctx, fn)
}
