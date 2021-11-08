package activerecord_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/activegraph/activegraph/activerecord"
	_ "github.com/activegraph/activegraph/activerecord/sqlite3"
)

type Hash map[string]interface{}

func TestActiveRecord_Insert(t *testing.T) {
	conn, err := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: ":memory:",
	})
	require.NoError(t, err)

	defer activerecord.RemoveConnection("primary")

	err = conn.Exec(
		context.TODO(), `
		CREATE TABLE authors (
			id		INTEGER NOT NULL,
			name	VARCHAR,

			PRIMARY KEY(id)
		);
		CREATE TABLE books (
			uid  		 INTEGER NOT NULL,
			author_id	 INTEGER,
			publisher_id INTEGER,
			year		 INTEGER,
			title		 VARCHAR,

			PRIMARY KEY(uid),
			FOREIGN KEY(author_id) REFERENCES author(id)
		);
		CREATE TABLE publishers (
			id      INTEGER NOT NULL,
			book_id INTEGER,
			
			PRIMARY KEY(id),
			FOREIGN KEY(book_id) REFERENCES book(id)
		);
		`,
	)
	require.NoError(t, err)

	Author := activerecord.New("author", func(r *activerecord.R) {
		r.HasMany("books")
	})

	Book := activerecord.New("book", func(r *activerecord.R) {
		r.PrimaryKey("uid")

		r.BelongsTo("author", func(assoc *activerecord.BelongsTo) {
			assoc.ForeignKey("author_id")
		})
		r.BelongsTo("publisher")
	})

	Publisher := activerecord.New("publisher", func(r *activerecord.R) {
		r.HasMany("book")
	})
	_ = Publisher

	author1 := Author.New(Hash{"name": "Herman Melville"})
	author2 := Author.New(Hash{"name": "Noah Harari"})

	book1 := Book.New(Hash{"title": "Bill Budd", "year": 1846, "author_id": 1})
	book2 := Book.New(Hash{"title": "Moby Dick", "year": 1851, "author_id": 1})
	book3 := Book.New(Hash{"title": "Omoo", "year": 1847, "author_id": 1})
	book4 := Book.New(Hash{"title": "Sapiens", "year": 2015, "author_id": 2})

	author1 = author1.Insert()
	author2 = author2.Insert()
	author1.Expect("Expecting author1 inserted")
	author2.Expect("Expecting author2 inserted")

	t.Logf("%s %s", author1, author2)

	book1 = book1.Insert()
	book1.Expect("Expecting book1 inserted")
	book2 = book2.Insert()
	book2.Expect("Expecting book2 inserted")
	book3 = book3.Insert()
	book3.Expect("Expecting book3 inserted")
	book4 = book4.Insert()
	book4.Expect("Expecting book4 inserted")

	t.Logf("%s %s %s %s", book1, book2, book3, book4)

	author := book1.Association("author").Unwrap()
	t.Logf("%s", author)

	authors, err := Author.All().ToA()
	require.NoError(t, err)
	t.Log(authors)

	bb := author1.Collection("books").ToA()
	t.Logf("%#v", bb)
	require.Len(t, bb, 3)

	// bb, _ = books.Where("year", 1851).ToA()
	// t.Log(bb)

	books := Book.All().Group("author_id", "year").Select("author_id", "year")

	t.Log(books)
	bb, err = books.ToA()
	t.Log(bb, err)

	bookAuthors := Book.Joins("author")
	bb, err = bookAuthors.ToA()
	t.Log(bb, err)
	t.Log(bb[0], bb[0].Association("author"))
	t.Log(bb[1], bb[1].Association("author"))
	t.Log(bb[2], bb[2].Association("author"))
	t.Log(bb[3], bb[3].Association("author"))
}

func TestActiveRecord_HasOne_AccessAssociation(t *testing.T) {
	conn, err := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: t.Name() + ".db",
	})
	require.NoError(t, err)

	defer os.Remove(t.Name() + ".db")
	defer activerecord.RemoveConnection("primary")

	err = conn.Exec(
		context.TODO(), `
		CREATE TABLE suppliers (
			id  		 INTEGER NOT NULL,
			name		 VARCHAR,

			PRIMARY KEY(id)
		);
		CREATE TABLE accounts (
			id      	INTEGER NOT NULL,
			supplier_id INTEGER,
			number		INTEGER,

			PRIMARY KEY(id),
			FOREIGN KEY(supplier_id) REFERENCES suppliers(id)
		);
		`,
	)
	require.NoError(t, err)

	Supplier := activerecord.New("supplier", func(r *activerecord.R) {
		r.HasOne("account")
	})

	Account := activerecord.New("account", func(r *activerecord.R) {
		r.BelongsTo("supplier")
	})

	suppliers, err := Supplier.InsertAll(
		Hash{"name": "Amazon"},
		Hash{"name": "Wallmart"},
	)
	require.NoError(t, err)

	accounts, err := Account.InsertAll(
		Hash{"number": 10, "supplier_id": suppliers[0].ID()},
		Hash{"number": 20, "supplier_id": suppliers[1].ID()},
	)
	require.NoError(t, err)

	t.Log(accounts)

	account := suppliers[0].Association("account").Unwrap()
	require.Equal(t, accounts[0].ID(), account.ID())
}
