package activerecord_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/activegraph/activegraph/activerecord"
	_ "github.com/activegraph/activegraph/activerecord/sqlite3"
	. "github.com/activegraph/activegraph/activesupport"
)

func initAuthorTable(t *testing.T, conn activerecord.Conn) {
	activerecord.Migrate(t.Name()+"_add_authors_table", func(m *activerecord.M) {
		m.CreateTable("authors", func(t *activerecord.Table) {
			t.String("name")
		})
	})
}

func initBookTable(t *testing.T, conn activerecord.Conn) {
	activerecord.Migrate(t.Name()+"_add_books_table", func(m *activerecord.M) {
		m.CreateTable("books", func(t *activerecord.Table) {
			t.Int64("year")
			t.String("title")
			t.References("authors")
			t.ForeignKey("authors")
		})
	})
}

func initProductTable(t *testing.T, conn activerecord.Conn) {
	activerecord.Migrate(t.Name()+"_add_products_table", func(m *activerecord.M) {
		m.CreateTable("products", func(t *activerecord.Table) {
			t.String("name")
		})
	})
}

func TestRelation_New(t *testing.T) {
	conn, _ := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: t.Name() + ".db",
	})

	defer os.Remove(t.Name() + ".db")
	defer activerecord.RemoveConnection("primary")

	initAuthorTable(t, conn)

	Author := activerecord.New("author")
	a := Author.New(Hash{"name": "Nassim Taleb"})
	a = a.Insert()

	require.NoError(t, a.Err())
	require.Equal(t, a.Unwrap().Attribute("name"), "Nassim Taleb")
}

func TestRelation_New_WithoutParams(t *testing.T) {
	conn, err := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: ":memory:",
	})

	require.NoError(t, err)

	defer activerecord.RemoveConnection("primary")
	initProductTable(t, conn)

	Product := activerecord.New("product")

	p := Product.New().Unwrap()
	require.NoError(t, p.AssignAttribute("name", "Holy Grail"))
}

func TestRelation_New_MultipleParams(t *testing.T) {
	conn, err := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: ":memory:",
	})
	require.NoError(t, err)

	defer activerecord.RemoveConnection("primary")
	initProductTable(t, conn)

	Product := activerecord.New("product", func(r *activerecord.R) {})
	p := Product.New(Hash{}, Hash{})

	require.Error(t, p.Err())

	err = &ErrMultipleVariadicArguments{Name: "params"}
	require.Equal(t, err.Error(), p.Err().Error())
}

func TestRelation_Limit(t *testing.T) {
	conn, _ := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: t.Name() + ".db",
	})

	defer os.Remove(t.Name() + ".db")
	defer activerecord.RemoveConnection("primary")

	initAuthorTable(t, conn)

	Author := activerecord.New("author")
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
	conn, err := activerecord.EstablishConnection(activerecord.DatabaseConfig{
		Adapter:  "sqlite3",
		Database: t.Name() + ".db",
	})
	require.NoError(t, err)

	defer os.Remove(t.Name() + ".db")
	defer activerecord.RemoveConnection("primary")

	initAuthorTable(t, conn)
	initBookTable(t, conn)

	Author := activerecord.New("author", func(r *activerecord.R) {
		r.HasMany("book")
	})

	Book := activerecord.New("book", func(r *activerecord.R) {
		r.BelongsTo("author")
	})

	err = activerecord.Transaction(context.TODO(), func() error {
		author := Author.Create(Hash{"name": "Max Tegmark"})

		book := author.AndThen(func(*activerecord.ActiveRecord) Result[*activerecord.ActiveRecord] {
			return Book.Create(Hash{"title": "Life 3.0", "year": 2017, "author_id": 1}).Result
		})

		return book.Err()
	})

	require.NoError(t, err)

	authors, err := Author.All().ToA()
	require.NoError(t, err)
	require.Len(t, authors, 1)

	book, err := Book.All().ToA()
	require.NoError(t, err)
	require.Len(t, book, 1)
}
