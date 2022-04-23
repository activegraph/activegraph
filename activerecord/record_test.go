package activerecord_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/activegraph/activegraph/activerecord"
	_ "github.com/activegraph/activegraph/activerecord/sqlite3"
	. "github.com/activegraph/activegraph/activesupport"
)

func TestActiveRecord_Insert(t *testing.T) {
	_, err := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: t.Name(),
	})
	require.NoError(t, err)

	defer os.Remove(t.Name())
	defer activerecord.RemoveConnection("primary")

	activerecord.Migrate(t.Name(), func(m *activerecord.M) {
		m.CreateTable("authors", func(t *activerecord.Table) {
			t.String("name")
		})
		m.CreateTable("books", func(t *activerecord.Table) {
			t.Int64("year")
			t.Int64("published_id")
			t.String("title")

			t.References("authors")
		})
		m.CreateTable("publishers", func(t *activerecord.Table) {
			t.Int64("book_id")
			t.References("books")
		})

		m.AddForeignKey("books", "authors")
		m.AddForeignKey("publishers", "books")
	})

	Author := activerecord.New("author", func(r *activerecord.R) {
		r.HasMany("books")
	})

	Book := activerecord.New("book", func(r *activerecord.R) {
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

	authors := Author.All().Expect("Expecting all authors")
	t.Log(authors)

	bb, err := author1.Collection("books").ToA()
	require.NoError(t, err)
	t.Logf("%#v", bb)
	require.Len(t, bb, 3)

	// bb, _ = books.Where("year", 1851).ToA()
	// t.Log(bb)

	books := Book.All().Unwrap().Group("author_id", "year").Select("author_id", "year")
	t.Log(books)

	bb, err = books.ToA()
	require.NoError(t, err)
	t.Log(bb)

	bookAuthors := Book.Joins("author")
	bb, err = bookAuthors.ToA()
	t.Log(bb, err)
	t.Log(bb[0], bb[0].Association("author"))
	t.Log(bb[1], bb[1].Association("author"))
	t.Log(bb[2], bb[2].Association("author"))
	t.Log(bb[3], bb[3].Association("author"))
}

func TestActiveRecord_HasOne_AccessAssociation(t *testing.T) {
	_, err := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: t.Name() + ".db",
	})
	require.NoError(t, err)

	defer os.Remove(t.Name() + ".db")
	defer activerecord.RemoveConnection("primary")

	activerecord.Migrate(t.Name(), func(m *activerecord.M) {
		m.CreateTable("suppliers", func(t *activerecord.Table) {
			t.String("name")
		})

		m.CreateTable("accounts", func(t *activerecord.Table) {
			t.Int64("number")
			t.References("suppliers") // Add "supplier_id" as fk
		})
	})

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
