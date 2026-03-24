package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	chaos "github.com/nikdoof/aaisp-chaos"
	"github.com/prometheus/client_golang/prometheus"
)

type mockClient struct {
	lines []chaos.BroadbandInfo
	err   error
}

func (m *mockClient) BroadbandInfo(_ context.Context) ([]chaos.BroadbandInfo, error) {
	return m.lines, m.err
}

func collectMetrics(t *testing.T, c prometheus.Collector) []prometheus.Metric {
	t.Helper()
	ch := make(chan prometheus.Metric, 20)
	c.Collect(ch)
	close(ch)
	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}
	return metrics
}

func TestCollectSuccess(t *testing.T) {
	mc := &mockClient{
		lines: []chaos.BroadbandInfo{
			{ID: 12345, Login: "test@a.1", Postcode: "SW1A1AA", TXRate: 80000000, TXRateAdjusted: 79000000, RXRate: 20000000, QuotaMonthly: 107374182400, QuotaRemaining: 53687091200},
		},
	}
	collector := broadbandCollector{client: mc, log: slog.Default()}

	metrics := collectMetrics(t, collector)
	// 6 metrics per quota line: line_info, quota_remaining, quota_total, tx_rate, tx_rate_adjusted, rx_rate
	if len(metrics) != 6 {
		t.Errorf("got %d metrics, want 6", len(metrics))
	}
}

func TestCollectUnlimitedLine(t *testing.T) {
	mc := &mockClient{
		lines: []chaos.BroadbandInfo{
			{ID: 57937, Login: "test@a.2", Postcode: "WA9 1SX", TXRate: 1000000000, TXRateAdjusted: 994690000, RXRate: 1000000000, QuotaMonthly: 0},
		},
	}
	collector := broadbandCollector{client: mc, log: slog.Default()}

	metrics := collectMetrics(t, collector)
	// 4 metrics for unlimited line: line_info, tx_rate, tx_rate_adjusted, rx_rate (no quota metrics)
	if len(metrics) != 4 {
		t.Errorf("got %d metrics, want 4", len(metrics))
	}
}

func TestCollectMultipleLines(t *testing.T) {
	mc := &mockClient{
		lines: []chaos.BroadbandInfo{
			{ID: 11111, Login: "test@a.1", Postcode: "SW1A1AA", TXRate: 80000000, TXRateAdjusted: 79000000, RXRate: 20000000, QuotaMonthly: 107374182400, QuotaRemaining: 53687091200},
			{ID: 22222, Login: "test@a.2", Postcode: "WA9 1SX", TXRate: 40000000, TXRateAdjusted: 39000000, RXRate: 10000000, QuotaMonthly: 53687091200, QuotaRemaining: 26843545600},
		},
	}
	collector := broadbandCollector{client: mc, log: slog.Default()}

	metrics := collectMetrics(t, collector)
	// 6 metrics per quota line × 2 lines
	if len(metrics) != 12 {
		t.Errorf("got %d metrics, want 12", len(metrics))
	}
}

func TestCollectError(t *testing.T) {
	mc := &mockClient{err: errors.New("API unavailable")}
	collector := broadbandCollector{client: mc, log: slog.Default()}

	metrics := collectMetrics(t, collector)
	if len(metrics) != 0 {
		t.Errorf("got %d metrics, want 0", len(metrics))
	}
}

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		level  string
		output string
	}{
		{"debug", "json"},
		{"info", "json"},
		{"warn", "console"},
		{"error", "console"},
		{"unknown", "json"}, // falls back to info
	}
	for _, tt := range tests {
		t.Run(tt.level+"/"+tt.output, func(t *testing.T) {
			log := setupLogger(tt.level, tt.output)
			if log == nil {
				t.Error("setupLogger() returned nil")
			}
		})
	}
}
