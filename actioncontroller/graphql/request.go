package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	graphql "github.com/vektah/gqlparser/v2/ast"
	grapherror "github.com/vektah/gqlparser/v2/gqlerror"
	graphparser "github.com/vektah/gqlparser/v2/parser"
	graphvalidate "github.com/vektah/gqlparser/v2/validator"
)

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
	schema *graphql.Schema        `json:"-"`
	query  *graphql.QueryDocument `json:"-"`
}

func (r *Request) Operation() string {
	panic("Not Implemented")
	// if r.document == nil {
	// 	return OperationUnknown
	// }
	// if len(r.document.Definitions) < 1 {
	// 	return OperationUnknown
	// }

	// opdef, ok := r.document.Definitions[0].(*qlast.OperationDefinition)
	// if !ok {
	// 	return OperationUnknown
	// }
	// return opdef.Operation
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

	query, e := graphparser.ParseQuery(&graphql.Source{Input: gr.Query})
	if e != nil {
		fmt.Println("???", grapherror.List{e}.Error())
		return nil, e
	}

	errs := graphvalidate.Validate(schema, query)
	if errs != nil {
		return nil, errs
	}

	// Copy the context of the HTTP request.
	gr.Header = r.Header.Clone()
	gr.ctx = r.Context()
	gr.query = query
	gr.schema = schema

	return gr, nil
}
