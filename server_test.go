package resly

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

func TestServer_ServeHTTPBasic(t *testing.T) {
	test := func(result []string) bool {
		s := Server{
			Queries: []FuncDef{
				NewFunc("posts", func(ctx context.Context) ([]string, error) {
					return result, nil
				}),
			},
		}

		var (
			url = "/graphql?query={posts}"

			rw = httptest.NewRecorder()
			r  = httptest.NewRequest(http.MethodGet, url, nil)
		)

		s.ServeHTTP(rw, r)

		return assert.Equal(t, http.StatusOK, rw.Code)
	}

	err := quick.Check(test, nil)
	assert.NoError(t, err)
}
