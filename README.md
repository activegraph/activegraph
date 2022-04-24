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
Consider the following example that creates two tables:
```go
activerecord.EstablishConnection(
    activerecord.DatabaseConfig{Adapter: "sqlite3", Database: "main.db",
}

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

## License

ActiveGraph is [MIT licensed](LICENSE).

[Tests]: https://github.com/activegraph/activegraph/workflows/Tests/badge.svg
[Documentation]: https://godoc.org/github.com/activegraph/activegraph?status.svg
