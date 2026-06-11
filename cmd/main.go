package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/logger"
	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/otel"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/config"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/handler"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/middleware"
	apexredis "github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/redis"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/service"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/spanner"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/storage"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {

	cfg, err := config.LoadUploadConfig()
	if err != nil {
		return err
	}

	logger := logger.New(cfg.Service(), cfg.Region(), cfg.AppEnv())
	bucketRegions := make([]string, 0, len(cfg.Buckets()))
	for region := range cfg.Buckets() {
		bucketRegions = append(bucketRegions, region)
	}
	sort.Strings(bucketRegions)

	logger.Info(
		context.Background(),
		"service.lifecycle.startup",
		"Upload API bootstrap started",
		slog.String("component", "bootstrap"),
		slog.Bool("audit", false),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	trcfg := otel.TracerConfig{
		AppEnv:      cfg.AppEnv(),
		ServiceName: cfg.Service(),
		ProjectID:   cfg.ProjectID(),
		Region:      cfg.Region(),
	}

	shutdown, err := otel.InitTracer(ctx, trcfg)
	if err != nil {
		return err
	}
	logger.Info(
		ctx,
		"service.lifecycle.telemetry_initialized",
		"OpenTelemetry initialized",
		slog.String("component", "telemetry"),
		slog.Bool("audit", false),
	)

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			logger.Error(
				context.Background(),
				"service.lifecycle.tracer_shutdown_failed",
				"Failed to shutdown tracer",
				slog.String("component", "telemetry"),
				slog.String("error", err.Error()),
				slog.Int("timeout_seconds", 5),
				slog.Bool("audit", false),
			)
			return
		}
		logger.Info(
			context.Background(),
			"service.lifecycle.tracer_shutdown_completed",
			"Tracer shutdown completed",
			slog.String("component", "telemetry"),
			slog.Int("timeout_seconds", 5),
			slog.Bool("audit", false),
		)
	}()

	spannerClient, err := spanner.NewClient(ctx, cfg.SpannerDatabase(), spanner.DefaultConfig())
	if err != nil {
		return err
	}
	defer spannerClient.Close()
	logger.Info(
		ctx,
		"service.lifecycle.spanner_initialized",
		"Spanner client initialized",
		slog.String("component", "bootstrap"),
		slog.Bool("audit", false),
	)

	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer gcsClient.Close()
	logger.Info(
		ctx,
		"service.lifecycle.storage_initialized",
		"Storage client initialized",
		slog.String("component", "bootstrap"),
		slog.Bool("audit", false),
	)

	redisClient, err := apexredis.NewClient(ctx, cfg.RedisAddr(), cfg.RedisPassword())
	if err != nil {
		return err
	}
	defer redisClient.Close()
	logger.Info(
		ctx,
		"service.lifecycle.redis_initialized",
		"Redis client initialized",
		slog.String("component", "bootstrap"),
		slog.String("addr", cfg.RedisAddr()),
		slog.Bool("audit", false),
	)

	uploadService := service.NewUploadService(logger, cfg, spannerClient, gcsClient, redisClient)
	logger.Info(
		ctx,
		"service.lifecycle.initialized",
		"Upload service initialized",
		slog.String("component", "bootstrap"),
		slog.Bool("audit", false),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.Healthz)
	mux.Handle("POST /upload", middleware.Authentication(logger)(handler.UploadHandler(logger, uploadService)))

	wrapped := otelhttp.NewHandler(
		middleware.Chain(
			mux,
			middleware.RequestID(logger),
			middleware.RequestLogging(logger),
		),
		"upload-api.http",
	)

	server := &http.Server{
		Addr:    cfg.Port(),
		Handler: wrapped,
	}

	logger.Info(
		ctx,
		"service.lifecycle.server_configured",
		"HTTP server configured",
		slog.String("component", "http_server"),
		slog.String("addr", cfg.Port()),
		slog.String("routes", "GET /healthz,POST /upload"),
		slog.Bool("request_id_middleware", true),
		slog.String("authentication_scope", "POST /upload"),
		slog.Bool("request_logging_middleware", true),
		slog.Bool("audit", false),
	)

	serverErrch := make(chan error, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info(
			ctx,
			"service.lifecycle.server_starting",
			"Starting HTTP server",
			slog.String("component", "http_server"),
			slog.String("addr", cfg.Port()),
			slog.Bool("audit", false),
		)
		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				logger.Info(
					context.Background(),
					"service.lifecycle.server_stopped",
					"HTTP server stopped accepting new requests",
					slog.String("component", "http_server"),
					slog.String("addr", cfg.Port()),
					slog.Bool("audit", false),
				)
				return
			}
			serverErrch <- err
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		logger.Info(
			ctx,
			"service.lifecycle.shutdown_initiated",
			"Shutdown signal received",
			slog.String("component", "lifecycle"),
			slog.String("reason", "signal"),
			slog.Bool("audit", false),
		)
	case err := <-serverErrch:
		runErr = err
		logger.Error(
			ctx,
			"service.lifecycle.server_failed",
			"HTTP server failed",
			slog.String("component", "http_server"),
			slog.String("addr", cfg.Port()),
			slog.String("error", err.Error()),
			slog.Bool("audit", false),
		)
	}

	stop()

	shutdownTimeout := 10 * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	logger.Info(
		context.Background(),
		"service.lifecycle.shutdown_started",
		"Starting HTTP server shutdown",
		slog.String("component", "http_server"),
		slog.String("addr", cfg.Port()),
		slog.Int("timeout_seconds", int(shutdownTimeout.Seconds())),
		slog.Bool("audit", false),
	)

	if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		logger.Error(
			context.Background(),
			"service.lifecycle.server_shutdown_failed",
			"Failed to shutdown HTTP server",
			slog.String("component", "http_server"),
			slog.String("addr", cfg.Port()),
			slog.String("error", err.Error()),
			slog.Int("timeout_seconds", int(shutdownTimeout.Seconds())),
			slog.Bool("audit", false),
		)
		if runErr == nil {
			runErr = fmt.Errorf("shutdown http server: %w", err)
		}
	} else {
		logger.Info(
			context.Background(),
			"service.lifecycle.shutdown_completed",
			"HTTP server shutdown gracefully",
			slog.String("component", "http_server"),
			slog.String("addr", cfg.Port()),
			slog.Int("timeout_seconds", int(shutdownTimeout.Seconds())),
			slog.Bool("audit", false),
		)
	}

	wg.Wait()
	logger.Info(
		context.Background(),
		"service.lifecycle.stopped",
		"Upload API stopped",
		slog.String("component", "lifecycle"),
		slog.Bool("clean_shutdown", runErr == nil),
		slog.Bool("audit", false),
	)
	return runErr
}
