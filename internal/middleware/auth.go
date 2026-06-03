package middleware

import (
	"log/slog"
	"net/http"

	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/logger"
	tier "github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/models"
	v1 "github.com/ctrlaltcloud008-hub/prj-apex-upload-service/api/v1"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/httputil"
)

func Authentication(logger *logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID, _ := RequestIDFromContext(r.Context())
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn(
					r.Context(),
					"auth.request.missing_header",
					"Rejected request without Authorization header",
					slog.String("request_id", requestID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Bool("audit", true),
				)
				httputil.WriteError(w, v1.NewMissingAuthorizationHeaderError())
				return
			}

			if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
				logger.Warn(
					r.Context(),
					"auth.request.invalid_format",
					"Rejected request with invalid Authorization header format",
					slog.String("request_id", requestID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Bool("audit", true),
				)
				httputil.WriteError(w, v1.NewInvalidAuthorizationHeaderError())
				return
			}

			token := authHeader[7:]
			if token == "" {
				logger.Warn(
					r.Context(),
					"auth.request.missing_token",
					"Rejected request with empty bearer token",
					slog.String("request_id", requestID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Bool("audit", true),
				)
				httputil.WriteError(w, v1.NewMissingBearerTokenError())
				return
			}

			clientRegionHint := r.Header.Get("X-Client-Region")

			userID := token
			ctx := WithUserID(r.Context(), userID)
			ctx = WithUserTier(ctx, tier.UserTierFree)
			ctx = WithClientRegionHint(ctx, clientRegionHint)
			logger.Info(
				ctx,
				"auth.request.authenticated",
				"Authenticated upload request",
				slog.String("request_id", requestID),
				slog.String("user_id", userID),
				slog.String("user_tier", string(tier.UserTierFree)),
				slog.String("client_region_hint", clientRegionHint),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Bool("audit", true),
			)

			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
