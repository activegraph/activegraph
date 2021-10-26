package graphql

import (
	"net/http"
	"strings"

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
	Write([]byte) error
}

type responseWriter struct {
	result []byte
}

func (rw *responseWriter) Write(b []byte) error {
	rw.result = b
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
		var grw responseWriter
		h.Serve(&grw, gr)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write(grw.result)
	}
}
