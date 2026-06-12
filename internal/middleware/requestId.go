package middleware

import (
	"log/slog"
	"net/http"

	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/logger"
	v1 "github.com/ctrlaltcloud008-hub/prj-apex-upload-service/api/v1"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/httputil"
	"github.com/google/uuid"
)

func RequestID(logger *logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.NewString()
			} else {
				if _, err := uuid.Parse(requestID); err != nil {
					logger.Warn(
						r.Context(),
						"auth.request.invalid_request_id",
						"Rejected request with invalid X-Request-ID header",
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.String("request_id", requestID),
						slog.Bool("audit", true),
					)
					httputil.WriteError(w, v1.NewInvalidRequestIDError(requestID))
					return
				}
			}

			w.Header().Set("X-Request-ID", requestID)

			ctx := WithRequestID(r.Context(), requestID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
