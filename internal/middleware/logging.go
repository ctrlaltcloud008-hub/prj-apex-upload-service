package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/logger"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func RequestLogging(logger *logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			requestID, _ := RequestIDFromContext(r.Context())
			logger.Info(
				r.Context(),
				"http.request",
				"Handled HTTP request",
				slog.String("request_id", requestID),
				slog.Bool("audit", false),
				slog.Group("httpRequest",
					slog.String("requestMethod", r.Method),
					slog.String("requestUrl", r.URL.String()),
					slog.String("protocol", r.Proto),
					slog.String("userAgent", r.UserAgent()),
					slog.String("remoteIp", r.RemoteAddr),
					slog.Int("status", rw.statusCode),
					slog.Float64("latency", time.Since(start).Seconds()),
				),
			)
		})
	}
}
