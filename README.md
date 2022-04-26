# [ActiveGraph](https://activegraph.github.io) &middot; [![Tests][Tests]](https://github.com/activegraph/activegraph) [![Documentation][Documentation]](https://godoc.org/github.com/activegraph/activegraph)

GraphQL-powered server framework for Go.

## Installation

You can install the latest version of the ActiveGraph module using `go mod`:
```bash
go get github.com/activegraph/activegraph
```

## Documentation

The [ActiveGraph Documentation](https://activegraph.github.io) contains additional
details on how to get started with GraphQL and ActiveGraph.

## Usage

Implementation of the ActiveGraph, comparing to other Go frameworks does not require
GraphQL schema declaration. Instead, it is highly anticipated to work with
business models of the service as API entities for GraphQL server.

### Active Records

You can provision database and create `ActiveRecord` from the database schema using
the `activerecord` package:
```go
activerecord.EstablishConnection(
    activerecord.DatabaseConfig{Adapter: "sqlite3", Database: "main.db"},
)

activerecord.Migrate("001_create_tables", func(m *activerecord.M) {
    m.CreateTable("authors", func(t *activerecord.Table) {
        t.String("name")
        t.DateTime("born_at")
    })

    m.CreateTable("books", func(t *activerecord.Table) {
        t.Int64("publisher_id")
        t.Int64("year")
        t.String("title")
        t.References("authors")
        t.ForeignKey("authors")
    })
})

// Declare records that reference created tables.
Book := activerecord.New("book", func(r *activerecord.R) {
    r.BelongsTo("author")
})

Author := activerecord.New("author", func(r *activerecord.R) {
    r.HasMany("books")
})
```

### Action Controller

You can create a GraphQL controller using `actioncontroller` package:
```go
AuthorControler := actioncontroller.New(func(c *actioncontroller.C) {
    // Generates "author(id: Int!)" query.
    c.Show(func(ctx *actioncontroller.Context) actioncontroller.Result {
        author := Author.Find(ctx.Params["id"])
        return actionview.NestedView(ctx, author)
    })

    // Generates "createAuthor(author: CreateAuthorInput!)" mutation.
    c.Create(func(ctx *actioncontroller.Context) actioncontroller.Result {
        author := Author.Create(ctx.Params.Get("author"))
        return actionview.NestedView(ctx, author)
    })

    // Generates "deleteAuthor(id: Int!)" mutation.
    c.Destroy(func(ctx *actioncontroller.Context) actioncontroller.Result {
        author := Author.Find(ctx.Params["id"])
        author = author.Delete()
        return actionview.NestedView(ctx, author)
    })
})
```

### Application

You can create a new application using `activegraph` package as following:
```py
app := activegraph.New(func(a *activegraph.A) {
    a.Resources(Author, AuthorController)
})

app.ListenAndServe() // Listens on localhost:3000 by default

// Use http://localhost:3000/graphql to access GrahiQL console.
```

## License

ActiveGraph is [MIT licensed](LICENSE).

[Tests]: https://github.com/activegraph/activegraph/workflows/Tests/badge.svg
[Documentation]: https://godoc.org/github.com/activegraph/activegraph?status.svg
