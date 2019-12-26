package httpserve

import (
	"net/http"
)

// TextHandler creates an HTTP handler that writes the given string
// and status as a response.
func TextHandler(status int, text string) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(status)
		rw.Header().Add("Content-Type", "text/plain")
		rw.Write([]byte(text))
	}
}
