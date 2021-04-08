package activerecord_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activerecord/sqlite3"
)

func TestRelation_JoinsOK(t *testing.T) {
	conn, err := sqlite3.Open(":memory:")
	require.NoError(t, err)

	defer conn.Close()

	err = conn.Exec(
		context.TODO(), `
		CREATE TABLE authors (
			id		INTEGER NOT NULL,
			name	VARCHAR,

			PRIMARY KEY(id)
		);
		CREATE TABLE books (
			id  		 INTEGER NOT NULL,
			author_id	 INTEGER,
			publisher_id INTEGER,
			title		 VARCHAR,

			PRIMARY KEY(id),
			FOREIGN KEY(author_id) REFERENCES author(id)
		);
		CREATE TABLE publishers (
			id      INTEGER NOT NULL,
			book_id INTEGER,
			name	VARCHAR,
			
			PRIMARY KEY(id),
			FOREIGN KEY(book_id) REFERENCES book(id)
		);
		`,
	)
	require.NoError(t, err)

	Author := activerecord.New("author", func(r *activerecord.R) {
		r.AttrString("name")
		r.HasMany("book")
	})

	Book := activerecord.New("book", func(r *activerecord.R) {
		r.AttrString("title")
		r.BelongsTo("author")
		r.BelongsTo("publisher")
	})

	Publisher := activerecord.New("publisher", func(r *activerecord.R) {
		r.AttrString("name")
		r.HasMany("book")
	})

	Author.Connect(conn)
	Book.Connect(conn)
	Publisher.Connect(conn)

	authors, err := Author.InsertAll(
		Hash{"name": "Herman Melville"}, Hash{"name": "Noah Harari"},
	)
	require.NoError(t, err)

	pubs, err := Publisher.InsertAll(
		Hash{"name": "MIT Press"}, Hash{"name": "CalTech Pub"},
	)
	require.NoError(t, err)

	books, err := Book.InsertAll(
		Hash{"title": "Bill Budd", "author_id": authors[0].ID(), "publisher_id": pubs[0].ID()},
		Hash{"title": "Moby Dick", "author_id": authors[0].ID(), "publisher_id": pubs[1].ID()},
		Hash{"title": "Omoo", "author_id": authors[0].ID()},
		Hash{"title": "Sapiens", "author_id": authors[1].ID(), "publisher_id": pubs[0].ID()},
	)
	require.NoError(t, err)

	books, err = Book.Joins("author", "publisher").ToA()
	require.NoError(t, err)
	require.Len(t, books, 3)

	// Close database connection and ensure the data is loaded.
	conn.Close()

	associations := map[int64]struct {
		authorId    int64
		publisherId int64
	}{
		1: {authorId: 1, publisherId: 1},
		2: {authorId: 1, publisherId: 2},
		4: {authorId: 2, publisherId: 1},
	}

	for _, book := range books {
		assocs, ok := associations[book.ID().(int64)]
		assert.True(t, ok)

		assert.Equal(t, assocs.authorId, book.Association("author").ID())
		assert.Equal(t, assocs.publisherId, book.Association("publisher").ID())
	}
}
