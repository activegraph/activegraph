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

	Author := activerecord.New("author", func(r *activerecord.R) {
		r.AttrString("name")
		r.HasMany("book")
	})

	Book := activerecord.New("book", func(r *activerecord.R) {
		r.PrimaryKey("uid")
		r.AttrInt("uid")
		r.AttrString("title")
		r.AttrInt("pages")
		r.BelongsTo("author") // author_id
	})

	Author.Connect(conn)
	Book.Connect(conn)

	author := Author.New(map[string]interface{}{"name": "Herman Melville"})
	author2 := Author.New(map[string]interface{}{"name": "Noa Harrary"})
	book := Book.New(map[string]interface{}{"title": "Moby Dick", "pages": 146, "author_id": 1})

	err = conn.Exec(
		context.TODO(), `
		CREATE TABLE authors (
			id		INTEGER NOT NULL,
			name	VARCHAR,

			PRIMARY KEY(id)
		);
		CREATE TABLE books (
			id  		INTEGER NOT NULL,
			author_id	INTEGER,
			pages		INTEGER,
			title		VARCHAR,

			PRIMARY KEY(id)
			FOREIGN KEY(author_id) REFERENCES author(id)
		);
		`,
	)
	require.NoError(t, err)

	author, err = author.Insert()
	require.NoError(t, err)
	t.Logf("%s", author)

	author2, err = author2.Insert()
	require.NoError(t, err)
	t.Logf("%s", author2)

	book, err = book.Insert()
	require.NoError(t, err)
	t.Logf("%s", book)

	author, err = book.Association("author")
	require.NoError(t, err)
	t.Logf("%s", author)

	authors, err := Author.All()
	require.NoError(t, err)
	t.Log(authors)

	err = book.Delete()
	require.NoError(t, err)
}
