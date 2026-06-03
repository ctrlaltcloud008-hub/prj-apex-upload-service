package utils

import (
	"log/slog"

	v1 "github.com/ctrlaltcloud008-hub/prj-apex-upload-service/api/v1"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/middleware"
)

func AuditAttrs(params middleware.Params, audit bool) []slog.Attr {
	return []slog.Attr{
		slog.String("request_id", params.RequestID),
		slog.String("user_id", params.UserID),
		slog.String("user_tier", string(params.UserTier)),
		slog.String("client_region_hint", params.ClientRegionHint),
		slog.Bool("audit", audit),
	}
}

func UploadAttrs(params middleware.Params, req *v1.UploadRequest, videoID string, audit bool) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("request_id", params.RequestID),
		slog.String("user_id", params.UserID),
		slog.String("user_tier", string(params.UserTier)),
		slog.String("client_region_hint", params.ClientRegionHint),
		slog.Bool("audit", audit),
	}
	if req != nil {
		attrs = append(attrs,
			slog.String("file_name", req.FileName),
			slog.Int64("file_size_bytes", req.FileSizeBytes),
			slog.String("content_type", req.ContentType),
		)
	}
	if videoID != "" {
		attrs = append(attrs, slog.String("video_id", videoID))
	}
	return attrs
}
