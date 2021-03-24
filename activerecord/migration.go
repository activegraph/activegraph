package activerecord

type Table struct {
	name string
}

func (t *Table) Column(name string) {
}

// Int adds a new int column to the named table.
func (t *Table) Int(name string) {
}

// String adds a new string column to the named table
func (t *Table) String(name string) {
}

type M struct {
}

func (m *M) CreateTable(name string, fn func(*Table)) {
}
