package v1

type UploadRequest struct {
	FileName      string `json:"file_name"`
	ContentType   string `json:"content_type"`
	FileSizeBytes int64  `json:"file_size_bytes"`
}

var AllowedContentTypes = map[string]bool{
	"video/mp4":        true,
	"video/webm":       true,
	"vidoe/quicktime":  true,
	"video/x-msvideo":  true,
	"video/x-matroska": true,
	"video/x-flv":      true,
	"video/3gpp":       true,
	"video/MP2T":       true,
}

func IsAllowedContentType(contentType string) bool {
	return AllowedContentTypes[contentType]
}

func IsValidFilename(fileName string) bool {

	if len(fileName) == 0 || len(fileName) > 255 {
		return false
	}

	for _, r := range fileName {
		if r == '/' || r == '\\' || r == 0 || (r < 32 && r != '\t') {
			return false
		}
	}

	return true
}
