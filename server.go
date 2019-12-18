package relsy

import (
	"net/http"

	graphqlhandler "github.com/graphql-go/handler"
)

type Server struct {
	graphql GraphQLCompiler
}

func (s *Server) ResolveType(typedef TypeDef) {
	if err := s.graphql.AddType(typedef); err != nil {
		panic(err)
	}
}

func (s *Server) ResolveQuery(funcdef FuncDef) {
	if err := s.graphql.AddQuery(funcdef); err != nil {
		panic(err)
	}
}

func (s *Server) ResolveMutation(funcdef FuncDef) {
	if err := s.graphql.AddMutation(funcdef); err != nil {
		panic(err)
	}
}

func (s *Server) ListenAndServe(addr string) {
	schema, err := s.graphql.Compile()
	if err != nil {
		panic(err)
	}

	h := graphqlhandler.New(&graphqlhandler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	http.Handle("/graphql", h)
	http.ListenAndServe(addr, nil)
}
