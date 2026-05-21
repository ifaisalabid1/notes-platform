package uploads

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

const MaxUploadSizeBytes int64 = 50 << 20

type ValidatedFile struct {
	OriginalFileName string
	ContentType      string
	SizeBytes        int64
	Data             []byte
	IsPDF            bool
}

var allowedContentTypes = map[string]bool{
	"application/pdf": true,

	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,

	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,

	"application/msword":       true,
	"application/vnd.ms-excel": true,
	"text/plain":               true,
}

func ValidateUploadedFile(file multipart.File, header *multipart.FileHeader) (ValidatedFile, error) {
	if file == nil {
		return ValidatedFile{}, errors.New("file is required")
	}

	if header == nil {
		return ValidatedFile{}, errors.New("file header is required")
	}

	originalFileName := sanitizeFileName(header.Filename)
	if originalFileName == "" {
		return ValidatedFile{}, errors.New("file name is required")
	}

	if header.Size <= 0 {
		return ValidatedFile{}, errors.New("file cannot be empty")
	}

	if header.Size > MaxUploadSizeBytes {
		return ValidatedFile{}, fmt.Errorf("file is too large, maximum allowed size is %d MB", MaxUploadSizeBytes/(1<<20))
	}

	data, err := io.ReadAll(io.LimitReader(file, MaxUploadSizeBytes+1))
	if err != nil {
		return ValidatedFile{}, fmt.Errorf("read uploaded file: %w", err)
	}

	if int64(len(data)) != header.Size {
		return ValidatedFile{}, errors.New("file size mismatch")
	}

	detectedMime := mimetype.Detect(data)
	contentType := detectedMime.String()

	if !allowedContentTypes[contentType] {
		return ValidatedFile{}, fmt.Errorf("unsupported file type: %s", contentType)
	}

	return ValidatedFile{
		OriginalFileName: originalFileName,
		ContentType:      contentType,
		SizeBytes:        int64(len(data)),
		Data:             data,
		IsPDF:            contentType == "application/pdf",
	}, nil
}

func Reader(data []byte) io.Reader {
	return bytes.NewReader(data)
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "/", "")

	return name
}
