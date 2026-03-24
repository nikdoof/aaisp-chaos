package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	chaos "github.com/nikdoof/aaisp-chaos"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	broadbandQuotaRemainingDesc = prometheus.NewDesc(
		"aaisp_broadband_quota_remaining",
		"Quota remaining in bytes",
		[]string{"line_id"},
		nil,
	)
	broadbandQuotaTotalDesc = prometheus.NewDesc(
		"aaisp_broadband_quota_total",
		"Quota total in bytes",
		[]string{"line_id"},
		nil,
	)
	broadbandLineInfoDesc = prometheus.NewDesc(
		"aaisp_broadband_line_info",
		"Static information about a broadband line",
		[]string{"line_id", "login", "postcode"},
		nil,
	)
	broadbandTXRateDesc = prometheus.NewDesc(
		"aaisp_broadband_tx_rate",
		"Maximum download rate in bits per second (AAISP transmit)",
		[]string{"line_id"},
		nil,
	)
	broadbandTXRateAdjustedDesc = prometheus.NewDesc(
		"aaisp_broadband_tx_rate_adjusted",
		"Adjusted download rate in bits per second after any throttling (AAISP transmit)",
		[]string{"line_id"},
		nil,
	)
	broadbandRXRateDesc = prometheus.NewDesc(
		"aaisp_broadband_rx_rate",
		"Maximum upload rate in bits per second (AAISP receive)",
		[]string{"line_id"},
		nil,
	)
	scrapeSuccessGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aaisp_scrape_success",
		Help: "Displays whether or not the AAISP API scrape was a success",
	})
)

// broadbandInfoFetcher is satisfied by *chaos.API and can be mocked in tests.
type broadbandInfoFetcher interface {
	BroadbandInfo(ctx context.Context) ([]chaos.BroadbandInfo, error)
}

type broadbandCollector struct {
	client broadbandInfoFetcher
	log    *slog.Logger
}

func (bc broadbandCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(bc, ch)
}

func (bc broadbandCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	lines, err := bc.client.BroadbandInfo(ctx)
	if err != nil {
		bc.log.Debug("error getting broadband info", "error", err)
		scrapeSuccessGauge.Set(0)
		return
	}
	scrapeSuccessGauge.Set(1)
	for _, line := range lines {
		lineID := strconv.Itoa(line.ID)
		ch <- prometheus.MustNewConstMetric(
			broadbandLineInfoDesc,
			prometheus.GaugeValue,
			1,
			lineID, line.Login, line.Postcode,
		)
		if line.QuotaMonthly > 0 {
			ch <- prometheus.MustNewConstMetric(
				broadbandQuotaRemainingDesc,
				prometheus.GaugeValue,
				float64(line.QuotaRemaining),
				lineID,
			)
			ch <- prometheus.MustNewConstMetric(
				broadbandQuotaTotalDesc,
				prometheus.CounterValue,
				float64(line.QuotaMonthly),
				lineID,
			)
		}
		ch <- prometheus.MustNewConstMetric(
			broadbandTXRateDesc,
			prometheus.GaugeValue,
			float64(line.TXRate),
			lineID,
		)
		ch <- prometheus.MustNewConstMetric(
			broadbandTXRateAdjustedDesc,
			prometheus.GaugeValue,
			float64(line.TXRateAdjusted),
			lineID,
		)
		ch <- prometheus.MustNewConstMetric(
			broadbandRXRateDesc,
			prometheus.GaugeValue,
			float64(line.RXRate),
			lineID,
		)
	}
}

func loggingMiddleware(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				remoteHost = r.RemoteAddr
			}
			log.Info("request",
				"proto", r.Proto,
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", remoteHost,
				"user_agent", r.Header.Get("User-Agent"),
			)
			next.ServeHTTP(w, r)
		})
	}
}

func usage(fs *flag.FlagSet) func() {
	return func() {
		o := fs.Output()
		fmt.Fprintf(o, "Usage:\n    %s ", os.Args[0])
		fs.VisitAll(func(f *flag.Flag) {
			s := fmt.Sprintf(" [-%s", f.Name)
			if arg, _ := flag.UnquoteUsage(f); len(arg) > 0 {
				s += " " + arg
			}
			s += "]"
			fmt.Fprint(o, s)
		})
		fmt.Fprint(o, "\n\nOptions:\n")
		fs.PrintDefaults()
		fmt.Fprint(o, "\nThe environment variables CHAOS_CONTROL_LOGIN and CHAOS_CONTROL_PASSWORD must be set.\n")
	}
}

func setupLogger(level, output string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if output == "console" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	return slog.New(handler)
}

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.Usage = usage(fs)
	var (
		listen    = fs.String("listen", ":8080", "listen `address`")
		logLevel  = fs.String("log.level", "info", "log `level`")
		logOutput = fs.String("log.output", "json", "log output `style` (json, console)")
	)
	fs.Parse(os.Args[1:])

	log := setupLogger(*logLevel, *logOutput)

	var (
		controlLogin    = os.Getenv("CHAOS_CONTROL_LOGIN")
		controlPassword = os.Getenv("CHAOS_CONTROL_PASSWORD")
	)
	switch {
	case controlLogin == "" && controlPassword == "":
		log.Error("CHAOS_CONTROL_LOGIN and CHAOS_CONTROL_PASSWORD must be set in the environment")
		os.Exit(1)
	case controlLogin == "":
		log.Error("CHAOS_CONTROL_LOGIN is not set")
		os.Exit(1)
	case controlPassword == "":
		log.Error("CHAOS_CONTROL_PASSWORD is not set")
		os.Exit(1)
	}

	collector := broadbandCollector{
		client: chaos.New(chaos.Auth{
			ControlLogin:    controlLogin,
			ControlPassword: controlPassword,
		}),
		log: log,
	}

	prometheus.MustRegister(collector)
	prometheus.MustRegister(scrapeSuccessGauge)

	mux := http.NewServeMux()
	mux.Handle("/metrics", loggingMiddleware(log)(promhttp.Handler()))

	server := &http.Server{
		Addr:    *listen,
		Handler: mux,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Info("shutting down server")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Error("server shutdown error", "error", err)
		}
	}()

	log.Info("listening", "addr", *listen)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
}
