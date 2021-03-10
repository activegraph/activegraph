package resly

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/graphql-go/graphql"
	qlast "github.com/graphql-go/graphql/language/ast"
	qlexpr "github.com/graphql-go/graphql/language/parser"
	qlsrc "github.com/graphql-go/graphql/language/source"
	"github.com/pkg/errors"
)

var (
	// ErrRequestTimeout is returned when request takes too much time
	// to process it.
	ErrRequestTimeout = errors.New("request processing timed out")
)

// textHandler creates an HTTP handler that writes the given string
// and status as a response.
func textHandler(status int, text string) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(status)
		rw.Header().Add("Content-Type", "text/plain")
		rw.Write([]byte(text))
	}
}

func parseURL(values url.Values) (*Request, error) {
	query := values.Get("query")
	if query == "" {
		return nil, errors.New("request is missing mandatory 'query' URL parameter")
	}

	var (
		vars    map[string]interface{}
		varsRaw = values.Get("variables")
	)

	if varsRaw != "" {
		if err := json.Unmarshal([]byte(varsRaw), &vars); err != nil {
			return nil, err
		}
	}

	return &Request{
		Query:         query,
		Variables:     vars,
		OperationName: values.Get("operationName"),
	}, nil
}

func parseForm(r *http.Request) (*Request, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	return parseURL(r.PostForm)
}

func parseGraphQL(r *http.Request) (*Request, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return &Request{Query: string(body)}, nil
}

// parseBody parses GraphQL request from the request JSON body.
func parseBody(r *http.Request) (*Request, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var gr Request
	err = json.Unmarshal(body, &gr)
	return &gr, err
}

// Request represents a GraphQL request received by server or to be sent by a client.
type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`

	// Header contains the request underlying HTTP header fields.
	//
	// These headers must be provided by the underlying HTTP request.
	Header http.Header

	// ctx represents the execution context of the request.
	ctx context.Context

	// Schema and parsed query as a GraphQL document.
	schema   *graphql.Schema `json:"-"`
	document *qlast.Document `json:"-"`
}

// Context returns the request's context.
//
// The returned context is always non-nil; it defaults to the background context.
func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

// TODO: create a shallow copy of the request.
func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

func parsePost(r *http.Request) (gr *Request, err error) {
	// For server requests body is always non-nil, but client request
	// can be passed here as well.
	if r.Body == nil {
		return nil, errors.Errorf("empty body for %s request", http.MethodPost)
	}

	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
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

// ParseRequest parses HTTP request and returns GraphQL request instance
// that contains all required parameters.
//
// This function supports requests provided as part of URL parameters,
// within body as pure GraphQL request, within body as JSON request, and
// as part of form request.
//
// Method ensures that query contains a valid GraphQL document and returns an error
// if it's not true.
func ParseRequest(r *http.Request, schema *graphql.Schema) (gr *Request, err error) {
	// Parse URL only when request is submitted with "GET" verb.
	switch r.Method {
	case http.MethodGet:
		gr, err = parseURL(r.URL.Query())
	case http.MethodPost:
		gr, err = parsePost(r)
	default:
		return gr, errors.Errorf("%s or %s verb is expected", http.MethodPost, http.MethodGet)
	}

	src := qlsrc.NewSource(&qlsrc.Source{
		Body: []byte(gr.Query), Name: "Request Query",
	})

	gr.document, err = qlexpr.Parse(qlexpr.ParseParams{Source: src})
	if err != nil {
		return nil, err
	}

	// Copy the context of the HTTP request.
	gr.Header = r.Header.Clone()
	gr.ctx = r.Context()
	gr.schema = schema

	return gr, nil
}

// ResponseWriter interface is used by a GraphQL handler to construct a response.
type ResponseWriter interface {
	Write(res *graphql.Result) error
}

type responseWriter struct {
	result *graphql.Result
}

func (rw *responseWriter) Write(res *graphql.Result) error {
	rw.result = res
	return nil
}

// Handler responds to a GraphQL request.
//
// Serve would write the reply to the ResponseWriter and then returns. Returning
// signals that the request is finished.
type Handler interface {
	Serve(ResponseWriter, *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (fn HandlerFunc) Serve(rw ResponseWriter, r *Request) {
	fn(rw, r)
}

// DefaultHandler is a default handler used by GraphQLhandler.
func DefaultHandler(rw ResponseWriter, r *Request) {
	result := graphql.Execute(graphql.ExecuteParams{
		Schema:        *r.schema,
		AST:           r.document,
		OperationName: r.OperationName,
		Args:          r.Variables,
		Context:       r.Context(),
	})

	rw.Write(result)
}

type Callback func(ResponseWriter, *Request)

type AroundCallback func(ResponseWriter, *Request, Handler)

// Server is a handler used to serve GraphQL requests.
type Server struct {
	// Name of the server. Will be used to emit metrics about resolvers.
	Name string

	// RequestTimeout is the maximum duration for handling the entire
	// request. When set to 0, request processing takes as much time
	// as needed.
	//
	// Default is no timeout.
	RequestTimeout time.Duration

	Types     []TypeDef
	Queries   []FuncDef
	Mutations []FuncDef

	callbacksAround []callbackAround
	callbacksBefore []callback
	callbacksAfter  []callback
}

type callback struct {
	op string
	cb Callback
}

type callbackAround struct {
	op string
	cb AroundCallback
}

const (
	// GraphQL operations.
	OperationQuery        = "query"        // a read-only fetch.
	OperationMutation     = "mutation"     // a write followed by fetch.
	OperationSubscription = "subscription" // unsupported yet.
)

// AppendAroundOp appends a callback after operations. See AroundCallback for parameter
// details.
//
// Use the around callback to wrap the GraphQL handler with a logic, e.g. execute
// the method in a read-only database transaction.
//
//	var s resly.Server
//	var txKey = struct{}{}
//
//	s.AppendAroundOp(rs.OperationQuery, func(rw rs.ResponseWriter, r *rs.Request, h rs.Handler) {
//		db.withRO(func(tx *db.Tx) error {
//			// Put transaction into the request context.
//			ctx = context.WithValue(r.Context(), txKey{}, tx)
//			// Serve GraphQL request with a modified context.
//			h.Serve(rw, r.WithContext(ctx))
//		})
//	})
//
func (s *Server) AppendAroundOp(op string, cb AroundCallback) {
	if cb == nil {
		panic("nil callback")
	}
	s.callbacksAround = append(s.callbacksAround, callbackAround{op, cb})
}

// AppendBeforeOp appends a callback before operations. See Callback for parameter
// details.
func (s *Server) AppendBeforeOp(op string, cb Callback) {
	if cb == nil {
		panic("nil callback")
	}
	s.callbacksBefore = append(s.callbacksBefore, callback{op, cb})
}

// AppendAfterOp appends a callback after operations. See AfterCallback for parameter
// details
func (s *Server) AppendAfterOp(op string, cb Callback) {
	if cb == nil {
		panic("nil callback")
	}
	s.callbacksAfter = append(s.callbacksAfter, callback{op, cb})
}

// HandleType adds given type definitions in the list of the types.
func (s *Server) HandleType(typedef ...TypeDef) *Server {
	s.Types = append(s.Types, typedef...)
	return s
}

// HandleQuery adds given function definition in the list of queries.
func (s *Server) HandleQuery(name string, fn interface{}) *Server {
	s.Queries = append(s.Queries, NewFunc(name, fn))
	return s
}

// HandleMutation adds given function definition in the list of mutations.
//
// Note, that GraphQL does not allow to use the same type as input and as
// an output within a single mutation. Therefore a common practice is to
// define input types as mutation input.
func (s *Server) HandleMutation(name string, fn interface{}) *Server {
	s.Mutations = append(s.Mutations, NewFunc(name, fn))
	return s
}

// CreateSchema returns compiled GraphQL schema from type and function
// definitions.
func (s *Server) CreateSchema() (schema graphql.Schema, err error) {
	var graphql GraphQL

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
	return graphql.CreateSchema()
}

func graphqlHandler(h Handler, schema graphql.Schema) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		if _, ok := r.URL.Query()["raw"]; !ok && strings.Contains(acceptHeader, "text/html") {
			handlePlayground(rw, r)
			return
		}

		gr, err := ParseRequest(r, &schema)
		if err != nil {
			h := textHandler(http.StatusBadRequest, err.Error())
			h.ServeHTTP(rw, r)
			return
		}

		// Serve the GraphQL request and write the result through HTTP.
		var grw responseWriter
		h.Serve(&grw, gr)

		b, err := json.Marshal(grw.result)
		if err != nil {
			h := textHandler(http.StatusInternalServerError, err.Error())
			h.ServeHTTP(rw, r)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write(b)
	}
}

// GraphQLHandler returns a new HTTP handler that attempts to parse GraphQL
// request from URL, body, or form and executes request using the specifies
// schema.
//
// On failed request parsing and execution method writes plain error message
// as a response.
func GraphQLHandler(schema graphql.Schema) http.HandlerFunc {
	return graphqlHandler(HandlerFunc(DefaultHandler), schema)
}

// HandleHTTP creates a new HTTP handler used to process GraphQL requests.
//
// This function registers all type and function definitions in the GraphQL
// schema. Produced schema will be used to resolve requests.
//
// On duplicate types, queries or mutations, function panics.
func (s *Server) HandleHTTP() http.Handler {
	// There is no reason to create a server that always returns errors.
	schema, err := s.CreateSchema()
	if err != nil {
		panic(err)
	}

	var handler http.Handler = GraphQLHandler(schema)
	if s.RequestTimeout != 0 {
		handler = http.TimeoutHandler(handler, s.RequestTimeout, ErrRequestTimeout.Error())
	}
	return handler
}
