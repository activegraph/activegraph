# Tutorial

> This tutorial is a step-by-step guide of building simple API server. If you're
> just getting started with GraphQL, we recommend going through [the GraphQL overview](https://graphql.org/learn).

## Step 1: Define a new model type

```go
type Post struct {
    ID       string `json:"id"`
    AuthorID string `json:"author_id"`
    Text     string `json:"text"`
}
```

## Step 2: Define function to query posts

```go
func queryPosts(ctx context.Context) ([]Post, error) {
    posts, err := db.Query(ctx)
    return posts, err
}
```

## Step 3: Configure a server

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

```go
import "net/http"

var s = resly.Server{ /* ... */ }

func main() {
    http.Handle("/graphql", s.MustCreateHandler())
    http.ListenAndServe(":8000", nil)
}
```
