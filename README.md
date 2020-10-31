# [Resly](https://resly.github.io) &middot; [![Tests][Tests]](https://github.com/resly/resly) [![Documentation][Documentation]](https://godoc.org/github.com/resly/resly)

GraphQL-powered server framework for Go.

## Installation

You can install the latest version of the Resly module using `go mod`:
```bash
go get github.com/resly/resly
```

## Documentation

The [Resly Documentation](https://resly.github.io) contains additional
details on how to get started with GraphQL and Resly.

## Usage

Implementation of the Resly, comparing to other Go frameworks does not require
GraphQL schema declaration. Instead, it is highly anticipated to work with
business models of the service as API entities for GraphQL server.

### Simple query
```go
import "context"
import "net/http"

import "github.com/resly/resly"


func main() {
    // Define a query function to retrieve "names".
    queryNames := func(ctx context.Context) ([]string, error) {
        return []string{"Steve", "Wozniak"}, nil
    }

    // Create Resly server with a single GraphQL query.
    s := resly.Server {
        Queries: []resly.FuncDef{resly.NewFunc("names", queryNames)},
    }

    // Serve GraphQL service at "localhost:8000/graphql" endpoint.
    http.Handle("/graphql", s)
    http.ListenAndServe(":8000", nil)
}
```

## License

Resly is [MIT licensed](LICENSE).

[Tests]: https://github.com/resly/resly/workflows/Tests/badge.svg
[Documentation]: https://godoc.org/github.com/resly/resly?status.svg
