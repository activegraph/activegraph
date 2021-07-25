package activerecord_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/activegraph/activegraph/activerecord"
	_ "github.com/activegraph/activegraph/activerecord/sqlite3"
	"github.com/activegraph/activegraph/activesupport"
)

func initAuthorTable(t *testing.T, conn activerecord.Conn) {
	err := conn.Exec(context.TODO(), `
		CREATE TABLE authors (
			id		INTEGER NOT NULL,
			name 	VARCHAR,

			PRIMARY KEY(id)
		);
	`)

	require.NoError(t, err)
}

func initBookTable(t *testing.T, conn activerecord.Conn) {
	err := conn.Exec(context.TODO(), `
		CREATE TABLE books (
			id 			INTEGER NOT NULL,
			author_id	INTEGER,
			year		INTEGER,
			title		VARCHAR,

			PRIMARY KEY(id),
			FOREIGN KEY(author_id) REFERENCES author(id)
		);
	`)
	require.NoError(t, err)
}

func TestRelation_New(t *testing.T) {
	Product := activerecord.New("product", func(r *activerecord.R) {
		r.AttrString("name")
	})

	p := Product.New(activesupport.Hash{"name": "Vacuum Cleaner"})
	require.NoError(t, p.Err())
	require.Equal(t, p.UnwrapRecord().Attribute("name"), "Vacuum Cleaner")
}

func TestRelation_New_WithoutParams(t *testing.T) {
	Product := activerecord.New("product", func(r *activerecord.R) {
		r.AttrString("name")
	})

	p := Product.New().UnwrapRecord()
	require.NoError(t, p.AssignAttribute("name", "Holy Grail"))
}

func TestRelation_New_MultipleParams(t *testing.T) {
	Product := activerecord.New("product", func(r *activerecord.R) {})
	p := Product.New(activesupport.Hash{}, activesupport.Hash{})

	require.Error(t, p.Err())

	err := &activesupport.ErrMultipleVariadicArguments{Name: "params"}
	require.Equal(t, err.Error(), p.Err().Error())
}

func TestRelation_Limit(t *testing.T) {
	activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: t.Name() + ".db",
	})

	defer os.Remove(t.Name() + ".db")
	defer activerecord.RemoveConnection("primary")

	Author := activerecord.New("author", func(r *activerecord.R) {
		r.AttrString("name")
	})

	initAuthorTable(t, Author.Connection())

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

func TestRelation_TransactionalInsert(t *testing.T) {
	activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: t.Name() + ".db",
	})

	defer os.Remove(t.Name() + ".db")
	defer activerecord.RemoveConnection("primary")

	Author := activerecord.New("author", func(r *activerecord.R) {
		r.AttrString("name")
		r.HasMany("book")
	})

	Book := activerecord.New("book", func(r *activerecord.R) {
		r.AttrString("title")
		r.AttrInt("year")
		r.BelongsTo("author")
	})

	initAuthorTable(t, Author.Connection())
	initBookTable(t, Book.Connection())

	activerecord.Transaction(context.TODO(), func() error {
		Author.Create(Hash{"name": "Max Tegmark"})
		Book.Create(Hash{"title": "Life 3.0", "year": 2017, "author_id": 1})
		return nil
	})

	authors, err := Author.All().ToA()
	require.NoError(t, err)
	require.Len(t, authors, 1)

	book, err := Book.All().ToA()
	require.NoError(t, err)
	require.Len(t, book, 1)
}
