package service

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/ctrlaltcloud008-hub/prj-apex-core-modules/pkg/models"
	"github.com/ctrlaltcloud008-hub/prj-apex-upload-service/internal/domain"
	"google.golang.org/api/iterator"
)

type idempotencyDecisionKind string

const (
	idempotencyDecisionMiss     idempotencyDecisionKind = "MISS"
	idempotencyDecisionReplay   idempotencyDecisionKind = "REPLAY"
	idempotencyDecisionMismatch idempotencyDecisionKind = "MISMATCH"
	idempotencyDecisionConsumed idempotencyDecisionKind = "CONSUMED"
)

type decision struct {
	Kind             idempotencyDecisionKind
	Record           *Record
	mismatchedFields []string
}

type Record struct {
	VideoID      string             `spanner:"video_id"`
	SourceBucket string             `spanner:"source_bucket"`
	SourceObject string             `spanner:"source_object"`
	MimeType     spanner.NullString `spanner:"mime_type"`
	FileSize     spanner.NullInt64  `spanner:"file_size"`
	Status       *models.Status     `spanner:"status"`
}

func CheckIdempotency(ctx context.Context, txn *spanner.ReadWriteTransaction, userID, requestID string, req domain.CreateUploadRequest) (decision, error) {

	stmt := spanner.Statement{
		SQL: `SELECT video_id, status, source_bucket, source_object, mime_type, file_size_bytes FROM videos
		WHERE user_id = @user_id AND request_id = @request_id
		ORDER BY created_at DESC
		LIMIT 1`,
		Params: map[string]interface{}{
			"user_id":    userID,
			"request_id": requestID,
		},
	}

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return decision{Kind: idempotencyDecisionMiss}, nil
	}

	if err != nil {
		return decision{}, fmt.Errorf("failed to query for idempotency: %w", err)
	}

	var record Record
	if err := row.Columns(&record.VideoID, &record.Status, &record.SourceBucket, &record.SourceObject, &record.MimeType, &record.FileSize); err != nil {
		return decision{}, fmt.Errorf("failed to read idempotency record: %w", err)
	}

	status := *record.Status

	if status != models.StatusUploading {
		return decision{
			Kind:   idempotencyDecisionConsumed,
			Record: &record,
		}, nil
	}

	mismatchedFields := mismatchFields(record, req)
	if len(mismatchedFields) > 0 {
		return decision{
			Kind:             idempotencyDecisionMismatch,
			Record:           &record,
			mismatchedFields: mismatchedFields,
		}, nil
	}

	return decision{
		Kind:   idempotencyDecisionReplay,
		Record: &record,
	}, nil
}

func mismatchFields(existing Record, req domain.CreateUploadRequest) []string {
	mismatchedFields := make([]string, 3)

	if !existing.MimeType.Valid || existing.MimeType.StringVal != req.ContentType {
		mismatchedFields = append(mismatchedFields, "content_type")
	}

	if !existing.FileSize.Valid || existing.FileSize.Int64 != req.FileSizeBytes {
		mismatchedFields = append(mismatchedFields, "file_size")
	}

	return mismatchedFields
}
