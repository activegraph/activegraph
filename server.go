package resly

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/graphql-go/graphql"

	"github.com/resly/resly/httpserve"
)

var (
	// ErrRequestTimeout is returned when request takes too much time
	// to process it.
	ErrRequestTimeout = errors.New("request processing timed out")
)

func parseURL(values url.Values) (gr GraphQLRequest, err error) {
	query := values.Get("query")
	if query == "" {
		return gr, errors.New("request is missing mandatory 'query' URL parameter")
	}

	var (
		vars    map[string]interface{}
		varsRaw = values.Get("variables")
	)

	if varsRaw != "" {
		if err = json.Unmarshal([]byte(varsRaw), &vars); err != nil {
			return gr, err
		}
	}

	return GraphQLRequest{
		Query:         query,
		Variables:     vars,
		OperationName: values.Get("operationName"),
	}, nil
}

func parseForm(r *http.Request) (gr GraphQLRequest, err error) {
	if err := r.ParseForm(); err != nil {
		return gr, err
	}
	return parseURL(r.PostForm)
}

func parseGraphQL(r *http.Request) (gr GraphQLRequest, err error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return gr, err
	}
	return GraphQLRequest{Query: string(body)}, nil
}

// parseBody parses GraphQL request from the request JSON body.
func parseBody(r *http.Request) (gr GraphQLRequest, err error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return gr, err
	}
	err = json.Unmarshal(body, &gr)
	return gr, err
}

// ParseRequest parses HTTP request and returns GraphQL request instance
// that contains all required parameters.
//
// This function supports requests provided as part of URL parameters,
// within body as pure GraphQL request, within body as JSON request, and
// as part of form request.
func ParseRequest(r *http.Request) (gr GraphQLRequest, err error) {
	// Parse URL only when request is submitted with "GET" verb.
	if r.Method == http.MethodGet {
		return parseURL(r.URL.Query())
	}
	if r.Method != http.MethodPost {
		return gr, errors.New("POST or GET verb is expected")
	}
	// For server requests body is always non-nil, but client request
	// can be passed here as well.
	if r.Body == nil {
		return gr, errors.New("empty body for POST request")
	}

	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return gr, err
	}

	switch contentType {
	case "application/graphql":
		return parseGraphQL(r)
	case "application/x-www-form-urlencoded":
		return parseForm(r)
	default:
		return parseBody(r)
	}
}

// Server is a handler used to serve GraphQL requests.
type Server struct {
	// RequestTimeout is the maximum duration for handling the entire
	// request. When set to 0, request processing takes as much time
	// as needed.
	//
	// Default is no timeout.
	RequestTimeout time.Duration

	Types     []TypeDef
	Queries   []FuncDef
	Mutations []FuncDef

	once    sync.Once
	handler http.Handler
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
// define input types as mutation input.
func (s *Server) AddMutation(funcdef FuncDef) {
	s.Mutations = append(s.Mutations, funcdef)
}

// compileSchema returns compiled GraphQL schema from type and function
// definitions.
func (s *Server) compileSchema() (schema graphql.Schema, err error) {
	var graphql GraphQLCompiler

	// Register all defined types and functions within a GraphQL compiler.
	for _, typedef := range s.Types {
		if err = graphql.AddType(typedef); err != nil {
			return schema, err
		}
	}
	for _, funcdef := range s.Queries {
		if err = graphql.AddQuery(funcdef); err != nil {
			return schema, err
		}
	}
	for _, funcdef := range s.Mutations {
		if err = graphql.AddMutation(funcdef); err != nil {
			return schema, err
		}
	}
	return graphql.Compile()
}

// GraphQLHandler returns a new HTTP handler that attempts to parse GraphQL
// request from URL, body, or form and executes request using the specifies
// schema.
//
// On failed request parsing and execution method writes plain error message
// as a response.
func GraphQLHandler(schema graphql.Schema) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		gr, err := ParseRequest(r)
		if err != nil {
			h := httpserve.TextHandler(http.StatusBadRequest, err.Error())
			h.ServeHTTP(rw, r)
			return
		}

		params := graphql.Params{
			Schema:         schema,
			RequestString:  gr.Query,
			VariableValues: gr.Variables,
			OperationName:  gr.OperationName,
			Context:        r.Context(),
		}

		b, err := json.Marshal(graphql.Do(params))
		if err != nil {
			h := httpserve.TextHandler(http.StatusInternalServerError, err.Error())
			h.ServeHTTP(rw, r)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write(b)
	}
}

// createHandler creates a new HTTP handler used to process GraphQL requests.
//
// This function registers all type and function definitions in the GraphQL
// schema. Produced schema will be used to resolve requests.
//
// On duplicate types, queries or mutations, function panics.
func (s *Server) createHandler() http.Handler {
	// There is no reason to create a server that always returns errors.
	schema, err := s.compileSchema()
	if err != nil {
		panic(err)
	}

	var handler http.Handler = GraphQLHandler(schema)
	if s.RequestTimeout != 0 {
		handler = http.TimeoutHandler(handler, s.RequestTimeout, ErrRequestTimeout.Error())
	}
	return handler
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	s.once.Do(func() { s.handler = s.createHandler() })
	s.handler.ServeHTTP(rw, r)
}
