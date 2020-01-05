# Tutorial

> This tutorial is a step-by-step guide of building simple API server. If you're
> just getting started with GraphQL, we recommend going through [the GraphQL overview](https://graphql.org/learn).

## Step 1: Define a new model type

Unlike other GraphQL frameworks, <b>Resly</b> does not forces to write GraphQL
schema separately from the Go types. JSON tags for each field of the Go type are
used to define names for GraphQL type fields.

```go
type Post struct {
    ID       string `json:"id"`
    AuthorID string `json:"author_id"`
    Text     string `json:"text"`
}
```

The `Post` type above generates the following GraphQL schema:
```graphql
type Post {
  id:        String!
  author_id: String!
  text:      String!
}
```

Unless you define field type as a pointer, it will generate _required_ field.

## Step 2: Define function to query posts

You can use existing functions to retrieve data from the the database, as long
as <b>Resly</b> does not inject any depenencies into the client code.

```go
func queryPosts(ctx context.Context) ([]Post, error) {
    posts, err := db.Query(ctx)
    return posts, err
}
```

## Step 3: Configure a server

The next step groups all components together, it connects types to the queries
used to retrieve them.

```go
var s = resly.Server {
    TypeDefs: []resly.TypeDef{
        // Register the GraphQL type called "Post".
        resly.NewType(Post{}, nil),
    },
    Queries: []resly.FuncDef{
        // Register query to retrieve a list of posts.
        resly.NewFunc("posts", queryPosts),
    },
}
```

## Step 4: Start the server

On the last step simply run the configured GraphQL server as HTTP server.

```go
import "net/http"

func main() {
    http.Handle("/graphql", s)
    http.ListenAndServe(":8000", nil)
}
```
