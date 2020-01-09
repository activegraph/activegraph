package resly

import (
	"context"
	"net/http"
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

// TracingHandler returns an http.Handler with opentracing context.
//
// The new Handler calls h.ServeHTTP to handle each request, it open
// span on each new request, logs query and variables, then closes
// span when handler finishes execution.
func TracingHandler(h http.Handler, tracer opentracing.Tracer) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		defer func() {
			h.ServeHTTP(rw, r)
		}()

		gr, err := ParseRequest(r)
		if err != nil {
			return
		}

		wireContext, err := tracer.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header),
		)
		if err != nil {
			return
		}

		span := opentracing.StartSpan(gr.OperationName, ext.RPCServerOption(wireContext))
		span.LogFields(
			log.String("query", gr.Query),
			log.Object("variables", gr.Variables),
		)

		defer span.Finish()

		ctx := opentracing.ContextWithSpan(r.Context(), span)
		r = r.WithContext(ctx)
	})
}
