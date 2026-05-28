package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/models"
	v1 "github.com/ctrlaltcloud008-hub/prj-apex-upload-service/api/v1"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/service"
)

func ClassifyError(err error) *v1.ErrorResponse {
	var idempotencyErr *service.IdempotencyMismatchError
	var requestErr *service.RequestIDAlreadyConsumedError

	switch {
	case errors.As(err, &idempotencyErr):
		return v1.NewErrorResponse(v1.StatusConflict, http.StatusConflict, "request ID conflicts with a different upload payload").WithReason(v1.ReasonIdempotencyMismatch).WithMetadata(map[string]string{"request_id": idempotencyErr.RequestID, "mismatched_fields": strings.Join(idempotencyErr.MismatchedFields, ",")}).WithInternal(err)
	case errors.As(err, &requestErr):
		return v1.NewErrorResponse(v1.StatusConflict, http.StatusConflict, "request ID already consumed").WithReason(v1.ReasonRequestIDAlreadyConsumed).WithMetadata(map[string]string{"request_id": requestErr.RequestID, "video_id": requestErr.VideoID, "status": requestErr.Status}).WithInternal(err)
	case errors.Is(err, models.ErrInvalidUserTier):
		return v1.NewErrorResponse(v1.StatusInvalidArgument, http.StatusBadRequest, "invalid user tier").WithInternal(err)
	case errors.Is(err, service.ErrFileTooLarge):
		return v1.NewErrorResponse(v1.StatusInvalidArgument, http.StatusRequestEntityTooLarge, "file too large").WithReason(v1.ReasonFileTooLarge).WithInternal(err)
	case errors.Is(err, service.ErrConcurrentUploadLimit):
		return v1.NewErrorResponse(v1.StatusResourceExhausted, http.StatusTooManyRequests, "concurrent upload limit reached").WithReason(v1.ReasonUploadLimitExceeded).WithInternal(err)
	case errors.Is(err, service.ErrHourlyUploadLimit):
		return v1.NewErrorResponse(v1.StatusResourceExhausted, http.StatusTooManyRequests, "hourly upload rate limit reached").WithReason(v1.ReasonHourlyRateExceeded).WithInternal(err)
	case errors.Is(err, service.ErrStorageQuotaExceeded):
		return v1.NewErrorResponse(v1.StatusPermissionDenied, http.StatusForbidden, "storage quota exceeded").WithReason(v1.ReasonStorageQuotaExceeded).WithInternal(err)
	case errors.Is(err, service.ErrUploadServiceUnavailable):
		return v1.NewErrorResponse(v1.StatusUnavailable, http.StatusServiceUnavailable, "upload service unavailable").WithInternal(err)
	case errors.Is(err, service.ErrSignedURLGeneration):
		return v1.NewErrorResponse(v1.StatusInternal, http.StatusInternalServerError, "failed to generate signed upload URL").WithInternal(err)
	default:
		return v1.NewErrorResponse(v1.StatusInternal, http.StatusInternalServerError, "failed to create upload").WithInternal(err)
	}

}
