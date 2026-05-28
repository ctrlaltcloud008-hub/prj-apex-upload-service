package utils

import (
	"log/slog"
	"net/http"

	v1 "github.com/ctrlaltcloud008-hub/prj-apex-upload-service/api/v1"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/middleware"
)

func LogAttrs(r *http.Request, req *v1.UploadRequest) []slog.Attr {
	params, isErr := middleware.ParamFromContext(r.Context())
	if !isErr {
		return []slog.Attr{}
	}

	attrs := []slog.Attr{
		slog.String("request_id", params.RequestID),
		slog.String("user_id", params.UserID),
		slog.String("user_tier", string(params.UserTier)),
		slog.String("client_region_hint", params.ClientRegionHint),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	}

	if req != nil {
		attrs = append(attrs,
			slog.String("file_name", req.FileName),
			slog.Int64("file_size", req.FileSizeBytes),
			slog.String("content_type", req.ContentType),
		)
	}

	return attrs
}
