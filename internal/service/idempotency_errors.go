package service

import (
	"errors"
	"fmt"
)

var (
	ErrIdempotencyMismatch      = errors.New("idempotency mismatch")
	ErrRequestIDAlreadyConsumed = errors.New("request id already consumer")
)

type IdempotencyMismatchError struct {
	RequestID        string
	MismatchedFields []string
}

func (e *IdempotencyMismatchError) Error() string {
	if len(e.MismatchedFields) == 0 {
		return fmt.Sprintf("%s: request_id=%q", ErrIdempotencyMismatch, e.RequestID)
	}
	return fmt.Sprintf("%s: request_id=%q, mismatched_fields=%v", ErrIdempotencyMismatch, e.RequestID, e.MismatchedFields)
}

func (e *IdempotencyMismatchError) Unwrap() error {
	return ErrIdempotencyMismatch
}

type RequestIDAlreadyConsumedError struct {
	RequestID string
	VideoID   string
	Status    string
}

func (e *RequestIDAlreadyConsumedError) Error() string {
	return fmt.Sprintf("%s: request_id=%q, video_id=%q, status=%q", ErrRequestIDAlreadyConsumed, e.RequestID, e.VideoID, e.Status)
}

func (e *RequestIDAlreadyConsumedError) Unwrap() error {
	return ErrRequestIDAlreadyConsumed
}
