package resly

import (
	"reflect"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
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
			received, err := newGraphQLType(tt.gotype, inObjectType, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, received)
		})
	}
}
