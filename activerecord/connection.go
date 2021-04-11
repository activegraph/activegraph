package activerecord

import (
	"fmt"
	"sync"
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
	pool     map[string]Conn
	mu       sync.RWMutex
}

func newConnectionHandler() *connectionHandler {
	return &connectionHandler{
		adapters: make(map[string]ConnectionAdapter),
		pool:     make(map[string]Conn),
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

func (h *connectionHandler) EstablishConnection(c DatabaseConfig) (Conn, error) {
	newConnection, ok := h.adapters[c.Adapter]
	if !ok {
		return nil, &ErrAdapterNotFound{Adapter: c.Adapter}
	}

	conn, err := newConnection(c)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if c.Name == "" {
		c.Name = primaryConnectionName
	}

	if _, dup := h.pool[c.Name]; dup {
		return nil, fmt.Errorf("connection %q established", c.Name)
	}

	h.pool[c.Name] = conn
	return conn, nil
}

func (h *connectionHandler) RetrieveConnection(name string) (Conn, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conn, ok := h.pool[name]
	if !ok {
		return nil, &ErrConnectionNotEstablished{Name: name}
	}
	return conn, nil
}

func (h *connectionHandler) RemoveConnection(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	conn, ok := h.pool[name]
	if !ok {
		return &ErrConnectionNotEstablished{Name: name}
	}

	delete(h.pool, name)
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
