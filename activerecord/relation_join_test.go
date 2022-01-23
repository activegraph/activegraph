package activerecord_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/activegraph/activegraph/activerecord"
	_ "github.com/activegraph/activegraph/activerecord/sqlite3"
	. "github.com/activegraph/activegraph/activesupport"
)

func TestRelation_JoinsOK(t *testing.T) {
	conn, err := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: t.Name() + ".db",
	})
	require.NoError(t, err)

	defer os.Remove(t.Name() + ".db")
	defer activerecord.RemoveConnection("primary")

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
		r.HasMany("book")
	})

	Book := activerecord.New("book", func(r *activerecord.R) {
		r.BelongsTo("author")
		r.BelongsTo("publisher")
	})

	Publisher := activerecord.New("publisher", func(r *activerecord.R) {
		r.HasMany("book")
	})

	authors, err := Author.InsertAll(
		Hash{"name": "Herman Melville"}, Hash{"name": "Noah Harari"},
	)
	require.NoError(t, err)
	require.Len(t, authors, 2)

	publishers, err := Publisher.InsertAll(
		Hash{"name": "MIT Press"}, Hash{"name": "CalTech Pub"},
	)
	require.NoError(t, err)
	require.Len(t, publishers, 2)

	books, err := Book.InsertAll(
		Hash{"title": "Bill Budd", "author_id": 1, "publisher_id": 1},
		Hash{"title": "Moby Dick", "author_id": 1, "publisher_id": 2},
		Hash{"title": "Omoo", "author_id": 1},
		Hash{"title": "Sapiens", "author_id": 2, "publisher_id": 1},
	)
	require.NoError(t, err)
	require.Len(t, books, 4)

	books, err = Book.Joins("author", "publisher").ToA()
	require.NoError(t, err)
	require.Len(t, books, 3)

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

		assert.Equal(t, assocs.authorId, book.Association("author").Unwrap().ID())
		assert.Equal(t, assocs.publisherId, book.Association("publisher").Unwrap().ID())
	}
}
