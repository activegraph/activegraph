package activetrace

import (
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/activegraph/activegraph"
)

// DefineMetricsFunc retruns a closure to measure the duration of the function.
func DefineMetricsFunc(subsystem string) activegraph.ClosureDef {
	requestDurationHistogramVec := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"query"},
	)

	return func(funcdef activegraph.FuncDef, in []reflect.Value) []reflect.Value {
		start := time.Now()
		defer func() {
			hist := requestDurationHistogramVec.WithLabelValues(funcdef.Name)
			hist.Observe(time.Since(start).Seconds())
		}()

		res, err := funcdef.Call(in)
		return []reflect.Value{
			reflect.ValueOf(res),
			reflect.ValueOf(err),
		}
	}
}
