package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/logger"
	v1 "github.com/ctrlaltcloud008-hub/prj-apex-upload-service/api/v1"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/domain"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/httputil"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/middleware"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/service"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/pkg/utils"
)

func UploadHandler(logger *logger.Logger, service service.UploadService) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var req v1.UploadRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&req); err != nil {
			logger.Warn(r.Context(), "upload.validation.invalid_json", "Reject upload request due to invalid JSON",
				slog.Bool("audit", false),
			)
			errPayload := v1.NewErrorResponse(v1.StatusInvalidArgument,
				http.StatusBadRequest, "Invalid JSON payload").WithMetadata(map[string]string{"error": err.Error()})
			httputil.WriteError(w, errPayload)
			return
		}

		if !v1.IsAllowedContentType(req.ContentType) {
			logger.Warn(r.Context(), "upload.validation.invalid_content_type", "Reject upload request due to invalid content type",
				slog.Bool("audit", false),
			)
			errPayload := v1.NewErrorResponse(v1.StatusInvalidArgument,
				http.StatusBadRequest, "Content type is not allowed").WithMetadata(map[string]string{"content_type": req.ContentType})
			httputil.WriteError(w, errPayload)
			return
		}

		if !v1.IsValidFilename(req.FileName) {
			logger.Warn(r.Context(), "upload.validation.invalid_filename", "Reject upload request due to invalid filename",
				slog.Bool("audit", false),
			)
			errPayload := v1.NewErrorResponse(v1.StatusInvalidArgument,
				http.StatusBadRequest, "Invalid file name").WithMetadata(map[string]string{"file_name": req.FileName})
			httputil.WriteError(w, errPayload)
			return
		}

		params, isErr := middleware.ParamFromContext(r.Context())
		if !isErr {
			logger.Warn(r.Context(), "upload.validation.missing_context_params", "Reject upload request due to missing context parameters",
				slog.Bool("audit", false),
			)
			errPayload := v1.NewErrorResponse(v1.StatusInternal, http.StatusInternalServerError, "Missing context parameters")
			httputil.WriteError(w, errPayload)
			return
		}

		logger.Info(r.Context(), "audit.upload.requested", "Start processing create upload request",
			utils.UploadAttrs(params, &req, "", true)...,
		)

		resp, err := service.CreateUpload(r.Context(), params, domain.CreateUploadRequest{
			Filename:      req.FileName,
			ContentType:   req.ContentType,
			FileSizeBytes: req.FileSizeBytes,
		})

		if err != nil {
			errPayload := ClassifyError(err)
			attrs := utils.UploadAttrs(params, &req, "", true)
			if errPayload.Reason != "" {
				attrs = append(attrs, slog.String("error_reason", string(errPayload.Reason)))
			}
			attrs = append(attrs, slog.String("error", err.Error()))
			logger.Error(r.Context(), "audit.upload.failed", "Failed to create upload", attrs...)
			httputil.WriteError(w, errPayload)
			return
		}

		logger.Info(r.Context(), "audit.upload.succeeded", "Successfully created upload",
			utils.UploadAttrs(params, &req, resp.VideoID, true)...,
		)

		respPayload := v1.UploadResponse{
			VideoID:          resp.VideoID,
			UploadURL:        resp.UploadURL,
			ExpiresAt:        resp.UploadURLExpiresAt.Unix(),
			MaxFileSizeBytes: resp.MaxFileSizeBytes,
		}
		httputil.WriteJSON(w, http.StatusCreated, respPayload)

	}

}
