// Package telemetry provides OpenTelemetry integration for the application.
// It supports traces and metrics export to OTLP and Prometheus backends.
package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/consts"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Default configuration values
const (
	defaultContextTimeout = 10 * time.Second
	defaultHTTPTimeout    = 10 * time.Second
	defaultPrometheusPort = 9090
)

// Config holds the telemetry configuration
type Config struct {
	// Enabled enables/disables telemetry
	Enabled bool `yaml:"enabled"`
	// ServiceName is the name of the service for telemetry
	ServiceName string `yaml:"service_name"`
	// OTLP configuration for trace export
	OTLP OTLPConfig `yaml:"otlp"`
	// Prometheus configuration for metrics export
	Prometheus PrometheusConfig `yaml:"prometheus"`
}

// OTLPConfig holds OTLP exporter configuration
type OTLPConfig struct {
	// Enabled enables OTLP trace export
	Enabled bool `yaml:"enabled"`
	// Endpoint is the OTLP collector endpoint (e.g., "localhost:4317")
	Endpoint string `yaml:"endpoint"`
	// Insecure disables TLS for the connection
	Insecure bool `yaml:"insecure"`
}

// PrometheusConfig holds Prometheus metrics configuration
type PrometheusConfig struct {
	// Enabled enables Prometheus metrics export
	Enabled bool `yaml:"enabled"`
	// Port is the port for the metrics HTTP server
	Port int `yaml:"port"`
}

// Telemetry manages OpenTelemetry providers and exporters
type Telemetry struct {
	config         Config
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	metricsServer  *http.Server
}

// New creates a new Telemetry instance with the given configuration
func New(cfg Config) (*Telemetry, error) {
	if !cfg.Enabled {
		logger.Info("Telemetry is disabled")
		return &Telemetry{config: cfg}, nil
	}

	// Set default service name
	if cfg.ServiceName == "" {
		cfg.ServiceName = consts.ServiceName
	}

	// Set default Prometheus port
	if cfg.Prometheus.Port == 0 {
		cfg.Prometheus.Port = defaultPrometheusPort
	}

	t := &Telemetry{config: cfg}

	// Create resource with service information
	// Use resource.New() to avoid schema URL conflicts between different semconv versions
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize tracer provider
	if err := t.initTracerProvider(res); err != nil {
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	// Initialize meter provider
	if err := t.initMeterProvider(res); err != nil {
		return nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("Telemetry initialized",
		zap.String("service_name", cfg.ServiceName),
		zap.Bool("otlp_enabled", cfg.OTLP.Enabled),
		zap.Bool("prometheus_enabled", cfg.Prometheus.Enabled),
	)

	return t, nil
}

// initTracerProvider initializes the tracer provider with OTLP exporter
func (t *Telemetry) initTracerProvider(res *resource.Resource) error {
	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithResource(res))

	// Add OTLP exporter if enabled
	if t.config.OTLP.Enabled && t.config.OTLP.Endpoint != "" {
		ctx, cancel := context.WithTimeout(context.Background(), defaultContextTimeout)
		defer cancel()

		exporterOpts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(t.config.OTLP.Endpoint),
		}
		if t.config.OTLP.Insecure {
			exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
		}

		exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
		if err != nil {
			return fmt.Errorf("failed to create OTLP trace exporter: %w", err)
		}

		opts = append(opts, sdktrace.WithBatcher(exporter))
		logger.Info("OTLP trace exporter initialized", zap.String("endpoint", t.config.OTLP.Endpoint))
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	t.tracerProvider = tp

	return nil
}

// initMeterProvider initializes the meter provider with Prometheus exporter
func (t *Telemetry) initMeterProvider(res *resource.Resource) error {
	var opts []sdkmetric.Option
	opts = append(opts, sdkmetric.WithResource(res))

	// Add Prometheus exporter if enabled
	if t.config.Prometheus.Enabled {
		exporter, err := prometheus.New()
		if err != nil {
			return fmt.Errorf("failed to create Prometheus exporter: %w", err)
		}
		opts = append(opts, sdkmetric.WithReader(exporter))

		// Start metrics HTTP server
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		t.metricsServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", t.config.Prometheus.Port),
			Handler:      mux,
			ReadTimeout:  defaultHTTPTimeout,
			WriteTimeout: defaultHTTPTimeout,
		}

		go func() {
			logger.Info("Starting Prometheus metrics server", zap.Int("port", t.config.Prometheus.Port))
			if err := t.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("Prometheus metrics server error", zap.Error(err))
			}
		}()
	}

	// Create meter provider
	mp := sdkmetric.NewMeterProvider(opts...)
	otel.SetMeterProvider(mp)
	t.meterProvider = mp

	return nil
}

// Shutdown gracefully shuts down all telemetry providers
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if !t.config.Enabled {
		return nil
	}

	logger.Info("Shutting down telemetry")

	// Shutdown tracer provider
	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown tracer provider", zap.Error(err))
		}
	}

	// Shutdown meter provider
	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown meter provider", zap.Error(err))
		}
	}

	// Shutdown metrics server
	if t.metricsServer != nil {
		if err := t.metricsServer.Shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown metrics server", zap.Error(err))
		}
	}

	return nil
}

// IsEnabled returns whether telemetry is enabled
func (t *Telemetry) IsEnabled() bool {
	return t.config.Enabled
}
