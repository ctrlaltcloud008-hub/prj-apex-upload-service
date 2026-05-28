package storage

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
)

type Client struct {
	gc *storage.Client
}

func NewClient(ctx context.Context) (*Client, error) {

	gc, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &Client{
		gc: gc,
	}, nil
}

func (c *Client) GenerateSignedResumableUploadInitURL(ctx context.Context, bucketName, objectName, contentType string, maxSize int64, expiryHours int) (string, time.Time, error) {
	if maxSize <= 0 {
		return "", time.Time{}, fmt.Errorf("max size must be greater than zero")
	}

	if expiryHours <= 0 {
		return "", time.Time{}, fmt.Errorf("expiry hours must be greater than zero")
	}

	expiresAt := time.Now().Add(time.Duration(expiryHours) * time.Hour)
	opts := &storage.SignedURLOptions{
		GoogleAccessID: "ci-cd-98@apex-494315.iam.gserviceaccount.com",
		Scheme:         storage.SigningSchemeV4,
		Method:         "POST",
		Expires:        expiresAt,
		ContentType:    contentType,
		Headers: []string{
			fmt.Sprint("Content-Length:0"),
			fmt.Sprintf("x-goog-resumable: start"),
		},
	}

	u, err := c.gc.Bucket(bucketName).SignedURL(objectName, opts)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate signed upload URL: %w", err)
	}

	return u, expiresAt, nil

}

func (c *Client) Close() error {
	return c.gc.Close()
}
