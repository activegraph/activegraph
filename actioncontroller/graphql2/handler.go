package graphql

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/activegraph/activegraph/activesupport"

	graphql "github.com/vektah/gqlparser/v2/ast"
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

// ResponseWriter interface is used by a GraphQL handler to construct a response.
type ResponseWriter interface {
	WriteData(k string, v interface{})
	WriteError(err error)
}

type responseWriter struct {
	data   activesupport.Hash
	errors []activesupport.Hash
}

func newResponseWriter() *responseWriter {
	return &responseWriter{data: make(activesupport.Hash)}
}

func (rw *responseWriter) WriteData(k string, v interface{}) {
	if v != nil {
		rw.data[k] = v
	}
}

func (rw *responseWriter) WriteError(err error) {
	if err != nil {
		rw.errors = append(rw.errors, activesupport.Hash{
			"message": err.Error(),
		})
	}
}

func (rw *responseWriter) MarshalJSON() ([]byte, error) {
	resp := activesupport.Hash{"data": nil}

	if len(rw.data) != 0 {
		resp["data"] = rw.data
	}
	if len(rw.errors) != 0 {
		resp["errors"] = rw.errors
	}
	return json.Marshal(resp)
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

// NewHandler returns a new HTTP handler that attempts to parse GraphQL
// request from URL, body, or form and executes request using the specifies
// schema.
//
// On failed request parsing and execution method writes plain error message
// as a response.
func NewHandler(h Handler, schema *graphql.Schema) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		if _, ok := r.URL.Query()["raw"]; !ok && strings.Contains(acceptHeader, "text/html") {
			handleGraphiQL(rw, r)
			return
		}

		gr, err := ParseRequest(r, schema)
		if err != nil {
			h := textHandler(http.StatusBadRequest, err.Error())
			h.ServeHTTP(rw, r)
			return
		}

		// Serve the GraphQL request and write the result through HTTP.
		grw := newResponseWriter()
		h.Serve(grw, gr)

		data, err := grw.MarshalJSON()
		if err != nil {
			// TODO: reply with 5xx
			panic(err)
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write(data)
	}
}
