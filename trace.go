package resly

import (
	"context"
	"reflect"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

func DefineTracingFunc(tracer opentracing.Tracer) ClosureDef {
	return func(funcdef FuncDef, in []reflect.Value) []reflect.Value {
		var (
			ctx context.Context = context.Background()
		)

		if len(in) > 1 {
			// Retrieve the context from the list of arguments.
			if funcCtx, ok := in[0].Interface().(context.Context); ok {
				ctx = funcCtx
			}
		}

		span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, tracer, funcdef.Name)
		defer span.Finish()

		res, err := funcdef.call(in)

		if err != nil {
			ext.Error.Set(span, true)
			span.LogFields(
				log.String("event", "error"),
				log.String("message", err.Error()),
			)
		}

		return []reflect.Value{
			reflect.ValueOf(res),
			reflect.ValueOf(err),
		}
	}
}
