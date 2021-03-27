package activerecord_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activerecord/sqlite3"
)

type Hash map[string]interface{}

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
		r.AttrInt("year")
		r.BelongsTo("author") // author_id
	})

	Author.Connect(conn)
	Book.Connect(conn)

	author1 := Author.New(Hash{"name": "Herman Melville"})
	author2 := Author.New(Hash{"name": "Noah Harari"})

	book1 := Book.New(Hash{"title": "Bill Budd", "year": 1846, "author_id": 1})
	book2 := Book.New(Hash{"title": "Moby Dick", "year": 1851, "author_id": 1})
	book3 := Book.New(Hash{"title": "Omoo", "year": 1847, "author_id": 1})
	book4 := Book.New(Hash{"title": "Sapiens", "year": 2015, "author_id": 2})

	err = conn.Exec(
		context.TODO(), `
		CREATE TABLE authors (
			id		INTEGER NOT NULL,
			name	VARCHAR,

			PRIMARY KEY(id)
		);
		CREATE TABLE books (
			uid  		INTEGER NOT NULL,
			author_id	INTEGER,
			year		INTEGER,
			title		VARCHAR,

			PRIMARY KEY(uid)
			FOREIGN KEY(author_id) REFERENCES author(id)
		);
		`,
	)
	require.NoError(t, err)

	author1, err = author1.Insert()
	require.NoError(t, err)
	author2, err = author2.Insert()
	require.NoError(t, err)

	t.Logf("%s %s", author1, author2)

	book1, err = book1.Insert()
	require.NoError(t, err)
	book2, err = book2.Insert()
	require.NoError(t, err)
	book3, err = book3.Insert()
	require.NoError(t, err)
	book4, err = book4.Insert()
	require.NoError(t, err)

	t.Logf("%s %s %s %s", book1, book2, book3, book4)

	author := book1.Association("author")
	t.Logf("%s", author)

	authors, err := Author.All().ToA()
	require.NoError(t, err)
	t.Log(authors)

	books := author1.Collection("book").Where("year > ?", 1846)
	bb, _ := books.ToA()
	require.Len(t, bb, 2)

	bb, _ = books.Where("year", 1851).ToA()
	t.Log(bb)

	books = Book.All().Group("author_id", "year").Select("author_id", "year")

	t.Log(books)
	bb, err = books.ToA()
	t.Log(bb, err)
}
