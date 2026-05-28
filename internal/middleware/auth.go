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
					"auth.missing_header",
					"Rejected request without Authorization header",
					slog.String("request_id", requestID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				httputil.WriteError(w, v1.NewMissingAuthorizationHeaderError())
				return
			}

			// Here you would typically validate the token or credentials in the authHeader.
			// For simplicity, we'll just check if it starts with "Bearer " and has a token.
			if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
				logger.Warn(
					r.Context(),
					"auth.invalid_header_format",
					"Rejected request with invalid Authorization header format",
					slog.String("request_id", requestID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				httputil.WriteError(w, v1.NewInvalidAuthorizationHeaderError())
				return
			}

			token := authHeader[7:]
			if token == "" {
				logger.Warn(
					r.Context(),
					"auth.missing_token",
					"Rejected request with empty bearer token",
					slog.String("request_id", requestID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				httputil.WriteError(w, v1.NewMissingBearerTokenError())
				return
			}

			clientRegionHint := r.Header.Get("X-Client-Region")

			// If the token is valid, you can set user information in the request context here.
			ctx := WithUserID(r.Context(), "exampleUser")
			ctx = WithUserTier(ctx, tier.UserTierFree)
			ctx = WithClientRegionHint(ctx, clientRegionHint)
			logger.Info(
				ctx,
				"auth.authenticated",
				"Authenticated upload request",
				slog.String("request_id", requestID),
				slog.String("user_id", "exampleUser"),
				slog.String("user_tier", string(tier.UserTierFree)),
				slog.String("client_region_hint", clientRegionHint),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)

			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
