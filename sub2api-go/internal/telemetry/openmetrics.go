package telemetry

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

var serverStart = time.Now()

// HTTPRequestsTotal is incremented by middleware (best-effort, lock-free).
var HTTPRequestsTotal uint64

// MetricsHandler exposes a minimal Prometheus/OpenMetrics text scrape without extra deps.
func MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		uptime := time.Since(serverStart).Seconds()
		n := atomic.LoadUint64(&HTTPRequestsTotal)

		body := fmt.Sprintf(
			"# HELP process_uptime_seconds Uptime of this process.\n"+
				"# TYPE process_uptime_seconds gauge\n"+
				"process_uptime_seconds %.3f\n"+
				"# HELP go_memstats_alloc_bytes Bytes allocated and still in use.\n"+
				"# TYPE go_memstats_alloc_bytes gauge\n"+
				"go_memstats_alloc_bytes %d\n"+
				"# HELP go_goroutines Number of goroutines.\n"+
				"# TYPE go_goroutines gauge\n"+
				"go_goroutines %d\n"+
				"# HELP sub2api_http_requests_total Total HTTP requests seen by metrics middleware.\n"+
				"# TYPE sub2api_http_requests_total counter\n"+
				"sub2api_http_requests_total %d\n",
			uptime,
			ms.Alloc,
			runtime.NumGoroutine(),
			n,
		)
		w.Header().Set("Content-Type", `text/plain; version=0.0.4; charset=utf-8`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
}

// IncHTTPRequest increments the scrape counter (call from Gin middleware).
func IncHTTPRequest() {
	atomic.AddUint64(&HTTPRequestsTotal, 1)
}
