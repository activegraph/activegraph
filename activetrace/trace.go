package activetrace

import (
	"context"
	"reflect"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"

	"github.com/activegraph/activegraph"
)

func DefineTracingFunc(tracer opentracing.Tracer) activegraph.ClosureDef {
	return func(funcdef activegraph.FuncDef, in []reflect.Value) []reflect.Value {
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

		res, err := funcdef.Call(in)

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

// TracingCallback returns an http.Handler with opentracing context.
//
// The new Handler calls h.ServeHTTP to handle each request, it open
// span on each new request, logs query and variables, then closes
// span when handler finishes execution.
func TracingCallback(tracer opentracing.Tracer) activegraph.AroundCallback {
	return func(rw activegraph.ResponseWriter, r *activegraph.Request, h activegraph.Handler) {
		wireContext, err := tracer.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header),
		)
		if err != nil {
			return
		}

		span := opentracing.StartSpan(r.OperationName, ext.RPCServerOption(wireContext))
		span.LogFields(
			log.String("query", r.Query),
			log.Object("variables", r.Variables),
		)

		defer span.Finish()

		ctx := opentracing.ContextWithSpan(r.Context(), span)
		r = r.WithContext(ctx)

		h.Serve(rw, r)
	}
}
