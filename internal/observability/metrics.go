package observability

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
)

// Metrics tracks operational metrics for the crawler.
type Metrics struct {
	// Request metrics
	RequestsTotal   atomic.Int64
	RequestsFailed  atomic.Int64
	RequestsRetried atomic.Int64

	// Response metrics
	ResponsesTotal atomic.Int64
	Responses2xx   atomic.Int64
	Responses3xx   atomic.Int64
	Responses4xx   atomic.Int64
	Responses5xx   atomic.Int64

	// Item metrics
	ItemsScraped atomic.Int64
	ItemsDropped atomic.Int64
	ItemsStored  atomic.Int64

	// Engine metrics
	ActiveWorkers   atomic.Int32
	QueueDepth      atomic.Int64
	BytesDownloaded atomic.Int64

	// Proxy metrics
	ProxyRotations atomic.Int64
	ProxyErrors    atomic.Int64

	logger *slog.Logger
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(logger *slog.Logger) *Metrics {
	return &Metrics{
		logger: logger.With("component", "metrics"),
	}
}

// ServeHTTP serves metrics in Prometheus text exposition format.
func (m *Metrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	metrics := []struct {
		name  string
		help  string
		value int64
	}{
		{"webstalk_requests_total", "Total requests made", m.RequestsTotal.Load()},
		{"webstalk_requests_failed_total", "Total failed requests", m.RequestsFailed.Load()},
		{"webstalk_requests_retried_total", "Total retried requests", m.RequestsRetried.Load()},
		{"webstalk_responses_total", "Total responses received", m.ResponsesTotal.Load()},
		{"webstalk_responses_2xx_total", "Total 2xx responses", m.Responses2xx.Load()},
		{"webstalk_responses_3xx_total", "Total 3xx responses", m.Responses3xx.Load()},
		{"webstalk_responses_4xx_total", "Total 4xx responses", m.Responses4xx.Load()},
		{"webstalk_responses_5xx_total", "Total 5xx responses", m.Responses5xx.Load()},
		{"webstalk_items_scraped_total", "Total items scraped", m.ItemsScraped.Load()},
		{"webstalk_items_dropped_total", "Total items dropped", m.ItemsDropped.Load()},
		{"webstalk_items_stored_total", "Total items stored", m.ItemsStored.Load()},
		{"webstalk_active_workers", "Currently active workers", int64(m.ActiveWorkers.Load())},
		{"webstalk_queue_depth", "Current URL queue depth", m.QueueDepth.Load()},
		{"webstalk_bytes_downloaded_total", "Total bytes downloaded", m.BytesDownloaded.Load()},
		{"webstalk_proxy_rotations_total", "Total proxy rotations", m.ProxyRotations.Load()},
		{"webstalk_proxy_errors_total", "Total proxy errors", m.ProxyErrors.Load()},
	}

	for _, metric := range metrics {
		fmt.Fprintf(w, "# HELP %s %s\n", metric.name, metric.help)
		fmt.Fprintf(w, "# TYPE %s counter\n", metric.name)
		fmt.Fprintf(w, "%s %d\n", metric.name, metric.value)
	}
}

// StartServer starts the metrics HTTP server.
func (m *Metrics) StartServer(port int, path string) error {
	mux := http.NewServeMux()
	mux.Handle(path, m)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	addr := fmt.Sprintf(":%d", port)
	m.logger.Info("metrics server starting", "addr", addr, "path", path)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			m.logger.Error("metrics server error", "error", err)
		}
	}()

	return nil
}

// Snapshot returns all metrics as a map.
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"requests_total":   m.RequestsTotal.Load(),
		"requests_failed":  m.RequestsFailed.Load(),
		"responses_total":  m.ResponsesTotal.Load(),
		"responses_2xx":    m.Responses2xx.Load(),
		"responses_4xx":    m.Responses4xx.Load(),
		"responses_5xx":    m.Responses5xx.Load(),
		"items_scraped":    m.ItemsScraped.Load(),
		"items_dropped":    m.ItemsDropped.Load(),
		"items_stored":     m.ItemsStored.Load(),
		"active_workers":   int64(m.ActiveWorkers.Load()),
		"queue_depth":      m.QueueDepth.Load(),
		"bytes_downloaded": m.BytesDownloaded.Load(),
	}
}
