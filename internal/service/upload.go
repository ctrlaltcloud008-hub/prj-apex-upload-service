package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/lifecycle"
	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/logger"
	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/models"
	entity "github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/models"
	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/quota"
	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/retry"
	spannerutils "github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/spanner"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/config"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/domain"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/middleware"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/storage"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrFileTooLarge             = errors.New("file too large")
	ErrConcurrentUploadLimit    = errors.New("concurrent upload limit reached")
	ErrHourlyUploadLimit        = errors.New("hourly upload limit reached")
	ErrStorageQuotaExceeded     = errors.New("storage quota exceeded")
	ErrSignedURLGeneration      = errors.New("signed URL generation failed")
	ErrUploadServiceUnavailable = errors.New("upload service unavailable")
)

const defaultRegion = "asia-south1"

const instrumentationName = "upload-api/internal/services"

type UploadService interface {
	CreateUpload(ctx context.Context, params middleware.Params, req domain.CreateUploadRequest) (*domain.CreateUploadResult, error)
}

type uploadService struct {
	logger  *logger.Logger
	cfg     *config.UploadConfig
	gcs     *storage.Client
	spanner *spanner.Client
}

func NewUploadService(logger *logger.Logger,
	cfg *config.UploadConfig,
	spanner *spanner.Client,
	gcs *storage.Client) UploadService {
	return &uploadService{
		logger:  logger,
		cfg:     cfg,
		spanner: spanner,
		gcs:     gcs,
	}
}

func (s *uploadService) CreateUpload(ctx context.Context, params middleware.Params, req domain.CreateUploadRequest) (*domain.CreateUploadResult, error) {

	ctx, span := otel.Tracer(instrumentationName).Start(ctx,
		"upload-api.create-upload.service",
		trace.WithAttributes(
			attribute.String("video.request_id", params.RequestID),
			attribute.String("video.user_id", params.UserID),
			attribute.String("video.user_tier", string(params.UserTier)),
			attribute.String("video.client_region_hint", params.ClientRegionHint),
			attribute.String("video.filename", req.Filename),
			attribute.String("video.content_type", req.ContentType),
			attribute.Int64("video.file_size_bytes", req.FileSizeBytes),
		),
	)
	defer span.End()

	logger := s.logger.WithSpanContext(ctx)
	logger.Info(
		ctx,
		"upload.processing.started",
		"Processing upload creation request",
		slog.String("request_id", params.RequestID),
		slog.String("user_id", params.UserID),
		slog.String("user_tier", string(params.UserTier)),
		slog.String("client_region_hint", params.ClientRegionHint),
		slog.String("filename", req.Filename),
		slog.String("content_type", req.ContentType),
		slog.Int64("file_size_bytes", req.FileSizeBytes),
		slog.Bool("audit", false),
	)

	limits, err := entity.GetTierLimits(params.UserTier)
	if err != nil {
		span.AddEvent("upload.limits.invalid_user_tier")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "invalid user tier")
		logger.Warn(
			ctx,
			"upload.validation.invalid_user_tier",
			"Rejected upload request for invalid user tier",
			slog.String("request_id", params.RequestID),
			slog.String("user_id", params.UserID),
			slog.String("user_tier", string(params.UserTier)),
			slog.String("error", err.Error()),
			slog.Bool("audit", false),
		)
		return nil, err
	}

	err = s.validateFileSizeForTier(limits, req.FileSizeBytes)
	if err != nil {
		span.AddEvent("upload.limits.file_size_rejected", trace.WithAttributes(
			attribute.Int64("video.max_file_size_bytes", limits.MaxFileSizeBytes),
		))
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "file size exceeds limit")
		logger.Warn(
			ctx,
			"upload.validation.file_too_large",
			"Rejected upload request that exceeds tier file size limit",
			slog.String("request_id", params.RequestID),
			slog.String("user_id", params.UserID),
			slog.Int64("file_size_bytes", req.FileSizeBytes),
			slog.Int64("max_file_size_bytes", limits.MaxFileSizeBytes),
			slog.String("error", err.Error()),
			slog.Bool("audit", false),
		)
		return nil, err
	}

	bucket, resolvedRegion := s.resolveRegionAndBucket(params.ClientRegionHint)
	videoID := s.generateVideoID()
	objectPath := s.buildObjectPath(params.UserID, videoID, req.Filename)
	span.SetAttributes(
		attribute.String("video.id", videoID),
		attribute.String("storage.bucket", bucket),
		attribute.String("storage.object", objectPath),
		attribute.String("video.resolved_region", resolvedRegion),
	)
	span.AddEvent("upload.storage.target_resolved")

	videoRecord := &entity.Video{
		VideoID:       videoID,
		UserID:        params.UserID,
		RequestID:     spanner.NullString{StringVal: params.RequestID, Valid: true},
		Status:        entity.StatusUploading,
		SourceBucket:  bucket,
		SourceObject:  objectPath,
		GCSGeneration: 0,
		MimeType:      spanner.NullString{StringVal: req.ContentType, Valid: req.ContentType != ""},
		FileSizeBytes: spanner.NullInt64{Int64: req.FileSizeBytes, Valid: req.FileSizeBytes > 0},
	}

	upload, err := s.executeSpannerTransaction(ctx, params.UserID, params.RequestID, req, limits, videoRecord)
	if err != nil {

		if errors.Is(err, ErrIdempotencyMismatch) || errors.Is(err, ErrRequestIDAlreadyConsumed) {
			span.AddEvent("upload.idempotency.rejected")
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "request id conflict")
			logger.Warn(
				ctx,
				"audit.upload.idempotency.conflict",
				"Rejected upload request due to request ID conflict",
				slog.String("request_id", params.RequestID),
				slog.String("user_id", params.UserID),
				slog.String("video_id", videoID),
				slog.String("error", err.Error()),
				slog.Bool("audit", true),
			)
			return nil, err
		}

		span.AddEvent("upload.transaction.failed")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to persist upload request")
		logger.Warn(
			ctx,
			"upload.transaction.failed",
			"Failed to persist upload request state",
			slog.String("request_id", params.RequestID),
			slog.String("user_id", params.UserID),
			slog.String("video_id", videoID),
			slog.String("bucket", bucket),
			slog.String("object_path", objectPath),
			slog.String("error", err.Error()),
			slog.Bool("audit", false),
		)
		return nil, err
	}

	if upload == nil {
		missingResultErr := fmt.Errorf("%w: missing stored video result", ErrUploadServiceUnavailable)
		span.AddEvent("upload.transaction.missing_result")
		span.RecordError(missingResultErr)
		span.SetStatus(otelcodes.Error, "missing stored video result")
		logger.Error(
			ctx,
			"upload.transaction.missing_result",
			"Upload transaction completed without stored video result",
			slog.String("request_id", params.RequestID),
			slog.String("user_id", params.UserID),
			slog.String("video_id", videoID),
			slog.Bool("audit", false),
		)
		return nil, missingResultErr
	}

	var uploadURL string
	var expiry time.Time

	err = retry.RetryWithBackoff(ctx, 3, func() error {
		generatedURL, generatedExpiry, err := s.gcs.GenerateSignedResumableUploadInitURL(ctx, bucket, objectPath, upload.MimeType.StringVal, upload.FileSize.Int64, int(limits.SignedURLExpirationHours))
		if err != nil {
			return err
		}
		uploadURL = generatedURL
		expiry = generatedExpiry
		return nil
	})

	if err != nil {
		span.AddEvent("upload.signed_url.generation_failed")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to generate signed upload url")
		logger.Error(
			ctx,
			"upload.storage.signed_url_failed",
			"Failed to generate signed upload URL",
			slog.String("request_id", params.RequestID),
			slog.String("user_id", params.UserID),
			slog.String("video_id", videoID),
			slog.String("bucket", bucket),
			slog.String("object_path", objectPath),
			slog.String("content_type", upload.MimeType.StringVal),
			slog.Int64("file_size_bytes", upload.FileSize.Int64),
			slog.String("error", err.Error()),
			slog.Bool("audit", false),
		)
		return nil, fmt.Errorf("%w: %w", ErrSignedURLGeneration, err)
	}

	logger.Info(
		ctx,
		"upload.storage.signed_url_generated",
		"Generated signed upload URL",
		slog.String("request_id", params.RequestID),
		slog.String("user_id", params.UserID),
		slog.String("video_id", upload.VideoID),
		slog.String("bucket", bucket),
		slog.String("object_path", objectPath),
		slog.Time("upload_url_expires_at", expiry),
		slog.Int64("max_file_size_bytes", limits.MaxFileSizeBytes),
		slog.Bool("audit", false),
	)

	return &domain.CreateUploadResult{
		VideoID:            videoID,
		UploadURL:          uploadURL,
		UploadURLExpiresAt: expiry,
		MaxFileSizeBytes:   limits.MaxFileSizeBytes,
	}, nil
}

func (s *uploadService) validateFileSizeForTier(limits entity.TierLimits, fileSize int64) error {

	if fileSize > limits.MaxFileSizeBytes {
		return fmt.Errorf("%w: max_file_size_bytes=%d", ErrFileTooLarge, limits.MaxFileSizeBytes)
	}

	return nil
}

func (s *uploadService) resolveRegionAndBucket(region string) (string, string) {
	if region == "" {
		region = defaultRegion
	}

	bucket, ok := s.cfg.Buckets()[region]
	if !ok {
		region = defaultRegion
		bucket = s.cfg.Buckets()[defaultRegion]
	}

	return bucket, region
}

func (s *uploadService) buildObjectPath(userID, videoID, fileName string) string {
	return fmt.Sprintf("%s/%s/%s", userID, videoID, fileName)
}

func (s *uploadService) normalizeObjectPath(bucket, sourceObject string) string {
	prefix := fmt.Sprintf("gs://%s/", bucket)
	return strings.TrimPrefix(sourceObject, prefix)
}

func (s *uploadService) generateVideoID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}

func (s *uploadService) executeSpannerTransaction(ctx context.Context, userID, requestID string, req domain.CreateUploadRequest, limits models.TierLimits, record *entity.Video) (*Record, error) {

	var result *Record
	logger := s.logger.WithSpanContext(ctx)

	_, err := spannerutils.RunRW(ctx, s.spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		decision, err := CheckIdempotency(ctx, txn, userID, requestID, req)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrUploadServiceUnavailable, err)
		}

		switch decision.Kind {
		case idempotencyDecisionReplay:
			logger.Info(ctx,
				"audit.upload.idempotency.replay",
				"Reused existing upload for request ID",
				slog.String("request_id", requestID),
				slog.String("user_id", userID),
				slog.String("video_id", decision.Record.VideoID),
				slog.String("status", string(*decision.Record.Status)),
				slog.Bool("audit", true),
			)
			result = &Record{
				VideoID:      decision.Record.VideoID,
				SourceBucket: decision.Record.SourceBucket,
				SourceObject: decision.Record.SourceObject,
				MimeType:     decision.Record.MimeType,
				FileSize:     decision.Record.FileSize,
				Status:       decision.Record.Status,
			}
			return nil
		case idempotencyDecisionMismatch:
			logger.Warn(ctx,
				"audit.upload.idempotency.mismatch",
				"Rejected upload request with mismatched payload",
				slog.String("request_id", requestID),
				slog.String("user_id", userID),
				slog.String("mismatched_fields", strings.Join(decision.mismatchedFields, ",")),
				slog.Bool("audit", true),
			)
			return &IdempotencyMismatchError{RequestID: requestID, MismatchedFields: decision.mismatchedFields}
		case idempotencyDecisionConsumed:
			logger.Warn(ctx,
				"audit.upload.idempotency.consumed",
				"Rejected upload request with request ID that has already been consumed",
				slog.String("request_id", requestID),
				slog.String("user_id", userID),
				slog.String("video_id", decision.Record.VideoID),
				slog.String("status", string(*decision.Record.Status)),
				slog.Bool("audit", true),
			)
			return &RequestIDAlreadyConsumedError{RequestID: requestID, VideoID: decision.Record.VideoID, Status: string(*decision.Record.Status)}
		}

		activeUploads, err := quota.CheckConcurrentLimit(ctx, txn, userID)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrUploadServiceUnavailable, err)
		}

		if activeUploads >= limits.MaxConcurrentUploads {
			logger.Warn(
				ctx,
				"upload.quota.concurrent_exceeded",
				"Rejected upload due to concurrent upload limit",
				slog.String("request_id", requestID),
				slog.String("user_id", userID),
				slog.Int64("active_uploads", activeUploads),
				slog.Int64("max_concurrent_uploads", limits.MaxConcurrentUploads),
				slog.Bool("audit", false),
			)
			return fmt.Errorf("%w: user_id=%q max_concurrent_uploads=%d", ErrConcurrentUploadLimit, userID, limits.MaxConcurrentUploads)
		}

		uploadsLastHour, err := quota.CheckHourlyRateLimit(ctx, txn, userID)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrUploadServiceUnavailable, err)
		}

		if uploadsLastHour >= limits.MaxUploadsPerHour {
			logger.Warn(
				ctx,
				"upload.quota.hourly_exceeded",
				"Rejected upload due to hourly upload limit",
				slog.String("request_id", requestID),
				slog.String("user_id", userID),
				slog.Int64("uploads_last_hour", uploadsLastHour),
				slog.Int64("max_uploads_per_hour", limits.MaxUploadsPerHour),
				slog.Bool("audit", false),
			)
			return fmt.Errorf("%w: user_id=%q max_uploads_per_hour=%d", ErrHourlyUploadLimit, userID, limits.MaxUploadsPerHour)
		}

		totalStorageUsed, err := quota.CheckStorageQuota(ctx, txn, userID)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrUploadServiceUnavailable, err)
		}

		totalStorageUsed += record.FileSizeBytes.Int64

		if totalStorageUsed > limits.StorageQuotaBytes {
			logger.Warn(
				ctx,
				"upload.quota.storage_exceeded",
				"Rejected upload due to storage quota limit",
				slog.String("request_id", requestID),
				slog.String("user_id", userID),
				slog.Int64("total_storage_used_bytes", totalStorageUsed),
				slog.Int64("storage_quota_bytes", limits.StorageQuotaBytes),
				slog.Bool("audit", false),
			)
			return fmt.Errorf("%w: user_id=%q storage_quota_bytes=%d", ErrStorageQuotaExceeded, userID, limits.StorageQuotaBytes)
		}

		if err := s.insertVideoRecord(ctx, txn, record); err != nil {
			return fmt.Errorf("%w: %w", ErrUploadServiceUnavailable, err)
		}

		if err := lifecycle.AppendLifecycleEvents(ctx, txn, record.VideoID, lifecycle.LifeCycleEventParams{
			ToStatus: models.StatusUploading,
			Actor:    "upload-api",
			Reason:   "upload_created",
		}); err != nil {
			return fmt.Errorf("%w: %w", ErrUploadServiceUnavailable, err)
		}

		if err := lifecycle.InsertVideoStageRecord(ctx, txn, lifecycle.StageRecordParams{
			VideoID: record.VideoID,
			Stage:   models.StatusUploading,
			Attempt: 1,
			StartedAt: spanner.NullTime{
				Time:  spanner.CommitTimestamp,
				Valid: true,
			},
			CompletedAt: spanner.NullTime{
				Valid: false,
			},
			DurationMs: spanner.NullInt64{
				Valid: false,
			},
			Outcome: spanner.NullString{
				Valid: false,
			},
			Actor: "upload-api",
			ErrorID: spanner.NullString{
				Valid: false,
			},
		}); err != nil {
			return fmt.Errorf("%w: %w", ErrUploadServiceUnavailable, err)
		}

		logger.Info(
			ctx,
			"upload.storage.record_persisted",
			"Persisted upload record and lifecycle state",
			slog.String("request_id", requestID),
			slog.String("user_id", userID),
			slog.String("video_id", record.VideoID),
			slog.String("bucket", record.SourceBucket),
			slog.String("object_path", record.SourceObject),
			slog.Int64("gcs_generation", record.GCSGeneration),
			slog.Bool("audit", false),
		)

		result = &Record{
			VideoID:      record.VideoID,
			SourceBucket: record.SourceBucket,
			SourceObject: record.SourceObject,
			MimeType:     record.MimeType,
			FileSize:     record.FileSizeBytes,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *uploadService) insertVideoRecord(ctx context.Context, txn *spanner.ReadWriteTransaction, videoRecord *models.Video) error {

	mutation := spanner.InsertOrUpdate("videos",
		[]string{"video_id", "user_id", "request_id", "status", "source_bucket", "source_object",
			"gcs_generation", "mime_type", "file_size_bytes", "created_at", "updated_at"},
		[]any{videoRecord.VideoID, videoRecord.UserID, videoRecord.RequestID, videoRecord.Status,
			videoRecord.SourceBucket, videoRecord.SourceObject,
			videoRecord.GCSGeneration, videoRecord.MimeType, videoRecord.FileSizeBytes, spanner.CommitTimestamp, spanner.CommitTimestamp},
	)

	return txn.BufferWrite([]*spanner.Mutation{mutation})
}
