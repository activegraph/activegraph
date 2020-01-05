# Tracing

<b>Resly</b> supports [Open Tracing](https://opentracing.io/) out of box. Tracer
is configured on the `Server` configuration step.

The example below shows how to configure [Jaeger](https://www.jaegertracing.io)
tracing for <b>Resly</b> server.

```go
import (
    "github.com/resly/resly"

    jaeger "github.com/uber/jaeger-client-go"
    jaegercfg "github.com/uber/jaeger-client-go/config"
)

func main() {
    // Setup Jaeger tracer configuration.
    cfg := jaegercfg.Configuration{
        ServiceName: "service_name",
        Sampler: &jaegercfg.SamplerConfig{
            Type: jaeger.SamplerTypeConst,
            Param: 1,
        },
        Reporter: &jaegercfg.ReporterConfig{
            LogSpans: true,
        },
    }

    // Create a new tracer instance.
    tracer, closer, err := jaeger.NewTracer(cfg)
    defer closer.Close()

    // Pass tracer to the server configuration.
    s := resly.Server {
        Tracer: tracer,
        Queries: []resly.FuncDef{ /* ... */ },
    }

    // ...
}
```
