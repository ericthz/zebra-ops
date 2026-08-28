package observability

import (
	"strings"
	"testing"
	"time"
)

func TestMetrics_IncAndRender(t *testing.T) {
	m := New()
	m.Inc("http_requests_total", "Total number of HTTP requests.", map[string]string{"method": "GET", "path": "/healthz", "status": "200"})
	m.Inc("http_requests_total", "Total number of HTTP requests.", map[string]string{"method": "GET", "path": "/healthz", "status": "200"})

	out := m.Render()
	if !strings.Contains(out, `http_requests_total{method="GET",path="/healthz",status="200"} 2`) {
		t.Fatalf("unexpected render output:\n%s", out)
	}
	if !strings.Contains(out, "# TYPE http_requests_total counter") {
		t.Fatalf("missing TYPE line:\n%s", out)
	}
}

func TestMetrics_ObserveHistogram(t *testing.T) {
	m := New()
	m.Observe("http_request_duration_seconds", "HTTP request latency.", map[string]string{"method": "POST", "path": "/api/chat"}, 50*time.Millisecond)

	out := m.Render()
	if !strings.Contains(out, "# TYPE http_request_duration_seconds histogram") {
		t.Fatalf("missing histogram TYPE line:\n%s", out)
	}
	if !strings.Contains(out, "_count") || !strings.Contains(out, "_sum") {
		t.Fatalf("missing histogram sum/count:\n%s", out)
	}
	if !strings.Contains(out, `le="0.1"} 1`) {
		t.Fatalf("expected bucket 0.1 to be 1:\n%s", out)
	}
}

func TestMetrics_DeterministicOrder(t *testing.T) {
	m := New()
	m.Inc("b", "b", map[string]string{"x": "1"})
	m.Inc("a", "a", map[string]string{"x": "1"})
	first := m.Render()
	second := m.Render()
	if first != second {
		t.Fatalf("render should be deterministic:\n%s\n---\n%s", first, second)
	}
}
