package resly

import (
	"context"
	"testing"
	"testing/quick"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type TestErrArg struct{}

func (TestErrArg) Validate() error {
	return errors.New("validation failed")
}

func TestFuncCall_ValidateError(t *testing.T) {
	fd := NewFunc("test", func(ctx context.Context, a TestErrArg) (string, error) {
		return "", nil
	})

	_, err := fd.CallUnbound(context.TODO(), map[string]interface{}{})
	assert.Error(t, errors.Cause(err), "validation failed")
}

type TestArg struct {
	Arg1 string `json:"arg1"`
}

func (TestArg) Validate() error {
	return nil
}

func TestFuncCall_ValidateOK(t *testing.T) {
	fd := NewFunc("test", func(ctx context.Context, a TestArg) (string, error) {
		return "", nil
	})

	call := func(arg1 string) bool {
		_, err := fd.CallUnbound(context.TODO(), map[string]interface{}{
			"input": map[string]interface{}{"arg1": arg1},
		})
		return err == nil
	}

	err := quick.Check(call, nil)
	assert.NoError(t, err)
}
