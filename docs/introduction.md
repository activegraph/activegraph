# Resly Documentation

<h4 style="color:grey">This page is an overview of the Resly documentation and related sources.</h4>

<b>Resly</b> is a Go library for building efficient API servers for GraphQL clients.
Learn what Resly is all about in the [tutorial](tutorial).

Resly provides ability to create a standalone GraphQL server, add a handler to existing
server or run it in serverless environment.

## Get Started

Implementation of the <b>Resly</b>, comparing to other Go frameworks does not require
GraphQL schema declaration. Instead, it is highly anticipated to works with business models
of the service as API entities for GraphQL server.

### Installation

You can retrieve the latest stable version of the library by running the following command:
```bash
go get github.com/resly/resly
```

### Hello World

The smallest <b>Resly</b> example looks like this:

```go
import (
    "context"
    "net/http"

    "github.com/resly/resly"
)

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

It starts GraphQL server that returns a list of strings on query: `query { names }`
