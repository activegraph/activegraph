package resly

import (
	"net/http"

	"github.com/graphql-go/handler"
)

// Server is a handler used to serve GraphQL requests.
//
// Example of basic handler with a complex type:
//
//	type PostInput struct {
//		AuthorID string `json:"author_id"`
//		Text     string `json:"text"`
//	}
//
//	type Post struct {
//		ID       string `json:"id"`
//		AuthorID string `json:"author_id"`
//		Text     string `json:"text"`
//	}
//
//	type Author struct {
//		ID   string `json:"id"`
//		Name string `json:"name"`
//	}
//
//	s := resly.Server{
//		Type: []resly.TypeDef{
//			resly.NewType(Post{}, resly.Funcs{
//				"author": func(ctx context.Context, p Post) (Author, error) {
//					// Fetch author.
//					return Author{}, nil
//				},
//			}),
//		},
//		Queries: []resly.FuncDefs{
//			resly.NewFunc("posts", func(ctx context.Context) ([]Post, error) {
//				// Fetch all posts.
//				return nil, nil
//			}),
//			resly.NewFunc("authors", func(ctx context.Context) ([]Author, error) {
//				// Fetch all authors.
//				return nil, nil
//			}),
//		},
//		Mutations: []resly.FuncDefs{
//			resly.NewFunc("createPost", func(ctx context.Context, p PostInput) (Post, error) {
//				// Create a new post.
//				return Post{}, nil
//			},
//		},
//	}
//
//	http.Handle("/graphql", s.MustCreateHandler())
//	http.ListenAndServe(":8080", nil)
//
type Server struct {
	Types     []TypeDef
	Queries   []FuncDef
	Mutations []FuncDef
}

// AddType adds given type definitions in the list of the types.
func (s *Server) AddType(typedef TypeDef) {
	s.Types = append(s.Types, typedef)
}

// AddQuery adds given function definition in the list of queries.
func (s *Server) AddQuery(funcdef FuncDef) {
	s.Queries = append(s.Queries, funcdef)
}

// AddMutation adds given function definition in the list of mutations.
//
// Note, that GraphQL does not allow to use the same type as input and as
// an output within a single mutation. Therefore a common practice is to
// define _input_ types as mutation input.
//
// Example of separation function input and output:
//
//	// PostInput lists arguments required to create a post.
//  //
//  // Such separation is required since user cannot generate identifier
//	// of the post, it's the responsibility of the server.
//	type PostInput struct {
//		AuthorID string `json:"author_id"`
//		Text     string `json:"text"`
//	}
//
//  // Post is the model of the post.
//	type Post struct {
//		ID       string `json:"id"`
//		AuthorID string `json:"author_id"`
//		Text     string `json:"text"`
//	}
//
//	mut := resly.NewFunc("createPost", func(ctx context.Context, p PostInput) (Post, error) {
//		// Insert a post into the database.
//		post := Post{
//			ID:       uuid.New(),
//			AuthorID: p.AuthorID,
//			Text:     p.Text,
//		}
//		db.Insert(post)
//		return post, nil
//	})
func (s *Server) AddMutation(funcdef FuncDef) {
	s.Mutations = append(s.Mutations, funcdef)
}

// CreateHandler creates a new HTTP handler used to process GraphQL requests.
//
// This function registers all type and function definitions in the GraphQL
// schema. Produced schema will be used to resolve requests.
//
// On duplicate types, queries or mutations, function returns an error.
func (s *Server) CreateHandler() (http.Handler, error) {
	var graphql GraphQLCompiler

	// Register all defined types and functions within a GraphQL compiler.
	for _, typedef := range s.Types {
		if err := graphql.AddType(typedef); err != nil {
			return nil, err
		}
	}
	for _, funcdef := range s.Queries {
		if err := graphql.AddQuery(funcdef); err != nil {
			return nil, err
		}
	}
	for _, funcdef := range s.Mutations {
		if err := graphql.AddMutation(funcdef); err != nil {
			return nil, err
		}
	}

	schema, err := graphql.Compile()
	if err != nil {
		return nil, err
	}

	return handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	}), nil
}

// MustCreateHandler executes CreateHandler, but panics, when the result
// produces an error.
func (s *Server) MustCreateHandler() http.Handler {
	h, err := s.CreateHandler()
	if err != nil {
		panic(err)
	}
	return h
}
