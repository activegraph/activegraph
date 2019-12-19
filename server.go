package resly

// Server is an HTTP handler used to serve GraphQL requests.
//
// Example of basic handler with a complex type:
//
//	s := resly.Server{
//		Type: []resly.TypeDef{
//			resly.NewType(Post{}, resly.Methods{
//				"author": func(ctx context.Context, p Post) (Author, error) {
//					// Fetch author.
//				},
//			}),
//		},
//		Query: map[string]FuncDef{
//		},
//	}
//
//	http.Handle("/graphql", s.Handler())
//
type Server struct {
	Type     map[string]TypeDef
	Query    map[string]FuncDef
	Mutation map[string]FuncDef
}

// type Server struct {
// 	graphql GraphQLCompiler
// }
//
// func (s *Server) ResolveType(typedef TypeDef) {
// 	if err := s.graphql.AddType(typedef); err != nil {
// 		panic(err)
// 	}
// }
//
// func (s *Server) ResolveQuery(funcdef FuncDef) {
// 	if err := s.graphql.AddQuery(funcdef); err != nil {
// 		panic(err)
// 	}
// }
//
// func (s *Server) ResolveMutation(funcdef FuncDef) {
// 	if err := s.graphql.AddMutation(funcdef); err != nil {
// 		panic(err)
// 	}
// }
//
// func (s *Server) ListenAndServe(addr string) {
// 	schema, err := s.graphql.Compile()
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	h := graphqlhandler.New(&graphqlhandler.Config{
// 		Schema:   &schema,
// 		Pretty:   true,
// 		GraphiQL: true,
// 	})
//
// 	http.Handle("/graphql", h)
// 	http.ListenAndServe(addr, nil)
// }
