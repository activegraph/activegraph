package activerecord_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activerecord/sqlite3"
)

func TestActiveRecord_Insert(t *testing.T) {
	conn, err := sqlite3.Open(":memory:")
	require.NoError(t, err)

	Book := activerecord.New(
		"book",
		activerecord.PrimaryKey{activerecord.IntAttr{Name: "uid"}},
		activerecord.StringAttr{Name: "title"},
		activerecord.IntAttr{Name: "pages"},
	)

	Book.Connect(conn)

	book := Book.New(map[string]interface{}{
		"title": "Moby Dick", "pages": 146,
	})

	if !book.HasAttribute("title") {
		t.Fatal("expected attribute 'title'")
	}

	err = conn.Exec(context.TODO(), "CREATE TABLE books (uid integer not null primary key, pages integer, title varchar);")
	require.NoError(t, err)

	book, err = book.Insert(context.TODO())
	require.NoError(t, err)
	t.Logf("ID is %v", book.ID())

	err = book.Delete(context.TODO())
	require.NoError(t, err)
}
