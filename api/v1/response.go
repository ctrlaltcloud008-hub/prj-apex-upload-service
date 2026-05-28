package v1

type UploadResponse struct {
	VideoID             string   `json:"video_id"`
	UploadURL           string   `json:"upload_url"`
	ExpiresAt           int64    `json:"expires_at"`
	MaxFileSizeBytes    int64    `json:"max_file_size_bytes"`
	AllowedContentTypes []string `json:"allowed_content_types"`
}
