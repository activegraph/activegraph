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

### Simple query
```go
import "context"
import "net/http"

import "github.com/activegraph/activegraph"


func main() {
    // Define a query function to retrieve "names".
    queryNames := func(ctx context.Context) ([]string, error) {
        return []string{"Steve", "Wozniak"}, nil
    }

    // Create ActiveGraph server with a single GraphQL query.
    s := activegraph.Server {
        Queries: []activegraph.FuncDef{activegraph.NewFunc("names", queryNames)},
    }

    // Serve GraphQL service at "localhost:8000/graphql" endpoint.
    http.Handle("/graphql", s)
    http.ListenAndServe(":8000", nil)
}
```

## License

ActiveGraph is [MIT licensed](LICENSE).

[Tests]: https://github.com/activegraph/activegraph/workflows/Tests/badge.svg
[Documentation]: https://godoc.org/github.com/activegraph/activegraph?status.svg
