package activerecord_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/activegraph/activegraph/activerecord"
	_ "github.com/activegraph/activegraph/activerecord/sqlite3"
)

func initTable(t *testing.T, conn activerecord.Conn) {
	err := conn.Exec(context.TODO(), `
		CREATE TABLE authors (
			id		INTEGER NOT NULL,
			name 	VARCHAR,

			PRIMARY KEY(id)
		);
	`)

	require.NoError(t, err)
}

func TestRelation_Limit(t *testing.T) {
	activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: ":memory:",
	})

	defer activerecord.RemoveConnection("primary")

	Author := activerecord.New("author", func(r *activerecord.R) {
		r.AttrString("name")

		//r.ConnectsTo(func(db *activerecord.ConnectsTo) {
		//	db.Writing("primary")
		//	db.Reading("primary_replicate")
		//})
	})

	initTable(t, Author.Connection())

	authors, err := Author.InsertAll(
		Hash{"name": "First"}, Hash{"name": "Second"},
		Hash{"name": "Third"}, Hash{"name": "Fourth"},
	)
	require.NoError(t, err)
	require.Len(t, authors, 4)

	authors, err = Author.Limit(3).ToA()
	require.NoError(t, err)
	require.Len(t, authors, 3)
}
