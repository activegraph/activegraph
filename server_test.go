package activegraph

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer_ServeHTTPBasic(t *testing.T) {
	a, z := int('a'), int('z')
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	genString := func() string {
		numChars := r.Intn(1 << 16)
		chars := make([]rune, numChars)
		for i := 0; i < numChars; i++ {
			chars[i] = rune(rand.Intn(z-a) + a)
		}
		return string(chars)
	}

	test := func(result []string) bool {
		s := Server{
			Name: genString(),
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

		s.HandleHTTP().ServeHTTP(rw, r)

		if !assert.Equal(t, http.StatusOK, rw.Code) {
			return false
		}

		var body struct {
			Data struct {
				Posts []string `json:"posts"`
			} `json:"data"`
		}

		err := json.Unmarshal(rw.Body.Bytes(), &body)
		if !assert.NoError(t, err) {
			return false
		}
		if len(result) == 0 && len(body.Data.Posts) == 0 {
			return true
		}

		return assert.Equal(t, result, body.Data.Posts)
	}

	err := quick.Check(test, nil)
	assert.NoError(t, err)
}
