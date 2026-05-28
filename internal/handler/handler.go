package handler

import (
	"encoding/json"
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
			logger.Warn(r.Context(), "upload.request_invalid_json", "Reject upload request due to invalid JSON", utils.LogAttrs(r, nil)...)
			errPayload := v1.NewErrorResponse(v1.StatusInvalidArgument,
				http.StatusBadRequest, "Invalid JSON payload").WithMetadata(map[string]string{"error": err.Error()})
			httputil.WriteError(w, errPayload)
			return
		}

		if !v1.IsAllowedContentType(req.ContentType) {
			logger.Warn(r.Context(), "upload.request_invalid_content_type", "Reject upload request due to invalid content type", utils.LogAttrs(r, nil)...)
			errPayload := v1.NewErrorResponse(v1.StatusInvalidArgument,
				http.StatusBadRequest, "Content type is not allowed").WithMetadata(map[string]string{"content_type": req.ContentType})
			httputil.WriteError(w, errPayload)
			return
		}

		if !v1.IsValidFilename(req.FileName) {
			logger.Warn(r.Context(), "upload.request_invalid_filename", "Reject upload request due to invalid filename", utils.LogAttrs(r, nil)...)
			errPayload := v1.NewErrorResponse(v1.StatusInvalidArgument,
				http.StatusBadRequest, "Invalid file name").WithMetadata(map[string]string{"file_name": req.FileName})
			httputil.WriteError(w, errPayload)
			return
		}

		params, isErr := middleware.ParamFromContext(r.Context())
		if !isErr {
			logger.Warn(r.Context(), "upload.request_missing_context_params", "Reject upload request due to missing context parameters", utils.LogAttrs(r, &req)...)
			errPayload := v1.NewErrorResponse(v1.StatusInternal, http.StatusInternalServerError, "Missing context parameters")
			httputil.WriteError(w, errPayload)
			return
		}

		logger.Info(r.Context(), "upload.create_upload_started", "Start processing create upload request", utils.LogAttrs(r, &req)...)

		resp, err := service.CreateUpload(r.Context(), params, domain.CreateUploadRequest{
			Filename:      req.FileName,
			ContentType:   req.ContentType,
			FileSizeBytes: req.FileSizeBytes,
		})

		if err != nil {
			errPayload := ClassifyError(err)
			logger.Error(r.Context(), "upload.create_upload_failed", "Failed to create upload", utils.LogAttrs(r, &req)...)
			httputil.WriteError(w, errPayload)
			return
		}

		logger.Info(r.Context(), "upload.create_upload_succeeded", "Successfully created upload", utils.LogAttrs(r, &req)...)

		respPayload := v1.UploadResponse{
			VideoID:          resp.VideoID,
			UploadURL:        resp.UploadURL,
			ExpiresAt:        resp.UploadURLExpiresAt.Unix(),
			MaxFileSizeBytes: resp.MaxFileSizeBytes,
		}
		httputil.WriteJSON(w, http.StatusCreated, respPayload)

	}

}
