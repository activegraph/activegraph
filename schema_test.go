package activegraph

import (
	"reflect"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGraphQLType_InputOK(t *testing.T) {
	tests := []struct {
		gotype reflect.Type
		want   graphql.Type
	}{
		{reflect.TypeOf(string("")), graphql.NewNonNull(graphql.String)},
		{reflect.TypeOf(new(string)), graphql.String},
		{reflect.TypeOf(int(0)), graphql.NewNonNull(graphql.Int)},
		{reflect.TypeOf(new(int)), graphql.Int},
		{reflect.TypeOf(float32(0)), graphql.NewNonNull(graphql.Float)},
		{reflect.TypeOf(new(float32)), graphql.Float},
		{reflect.TypeOf(float64(0)), graphql.NewNonNull(graphql.Float)},
		{reflect.TypeOf(new(float64)), graphql.Float},
		{reflect.TypeOf([]string{}), graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
		{reflect.TypeOf([]int{}), graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.Int)))},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			received, err := newType(tt.gotype, inObjectType, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, received)
		})
	}
}

func TestNewQueryArgs_OK(t *testing.T) {
	type QueryArgs map[string]graphql.Type

	tests := []struct {
		gotype interface{}
		want   QueryArgs
	}{
		{struct{ Name string }{}, QueryArgs{"name": graphql.NewNonNull(graphql.String)}},
		{struct {
			Key   int
			Value *uint64
		}{}, QueryArgs{"key": graphql.NewNonNull(graphql.Int), "value": graphql.Int}},
		{struct{ UserName *string }{}, QueryArgs{"userName": graphql.String}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			conf, err := newQueryArgs(reflect.TypeOf(tt.gotype), make(map[string]graphql.Type))
			require.NoError(t, err)

			args := make(QueryArgs, len(conf))
			for name, arg := range conf {
				args[name] = arg.Type
			}

			assert.Equal(t, tt.want, args)
		})
	}
}
