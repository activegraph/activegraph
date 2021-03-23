package activerecord

type TableDefinition struct {
	name string
}

func (d *TableDefinition) Column(name string) {
}

// Int adds a new int column to the named table.
func (d *TableDefinition) Int(name string) {
}

// String adds a new string column to the named table
func (d *TableDefinition) String(name string) {
}

type Migration struct {
}

func (m *Migration) CreateTable(name string, fn func(TableDefinition)) {
}
