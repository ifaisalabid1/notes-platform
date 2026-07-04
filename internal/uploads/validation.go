package uploads

import (
	"archive/zip"
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
const MaxZipExtractedSizeBytes int64 = 100 << 20
const MaxZipImageCount = 100

type ValidatedFile struct {
	OriginalFileName string
	ContentType      string
	SizeBytes        int64
	Data             []byte
	IsPDF            bool
	IsZIP            bool
}

var allowedContentTypes = map[string]bool{
	"application/pdf": true,

	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,

	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,

	"application/msword":           true,
	"application/vnd.ms-excel":     true,
	"text/plain":                   true,
	"application/zip":              true,
	"application/x-zip":            true,
	"application/x-zip-compressed": true,
}

var allowedImageContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
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
		IsZIP:            contentType == "application/zip" || contentType == "application/x-zip" || contentType == "application/x-zip-compressed",
	}, nil
}

func ValidateZipImageFiles(zipFile ValidatedFile) ([]ValidatedFile, error) {
	if !zipFile.IsZIP {
		return nil, errors.New("uploaded file is not a ZIP archive")
	}

	reader, err := zip.NewReader(bytes.NewReader(zipFile.Data), int64(len(zipFile.Data)))
	if err != nil {
		return nil, errors.New("invalid ZIP archive")
	}

	files := make([]ValidatedFile, 0)
	var totalExtractedBytes int64

	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}

		fileName := sanitizeFileName(entry.Name)
		if shouldSkipZipEntry(entry.Name, fileName) {
			continue
		}

		if fileName == "" {
			return nil, errors.New("ZIP contains an image without a valid file name")
		}

		if len(files) >= MaxZipImageCount {
			return nil, fmt.Errorf("ZIP contains too many images, maximum allowed is %d", MaxZipImageCount)
		}

		if entry.UncompressedSize64 == 0 {
			return nil, fmt.Errorf("ZIP image %s is empty", fileName)
		}

		if entry.UncompressedSize64 > uint64(MaxUploadSizeBytes) {
			return nil, fmt.Errorf("ZIP image %s is too large, maximum allowed size per image is %d MB", fileName, MaxUploadSizeBytes/(1<<20))
		}

		totalExtractedBytes += int64(entry.UncompressedSize64)
		if totalExtractedBytes > MaxZipExtractedSizeBytes {
			return nil, fmt.Errorf("ZIP extracted image data is too large, maximum allowed is %d MB", MaxZipExtractedSizeBytes/(1<<20))
		}

		entryReader, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("open ZIP image %s: %w", fileName, err)
		}

		data, readErr := io.ReadAll(io.LimitReader(entryReader, MaxUploadSizeBytes+1))
		closeErr := entryReader.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read ZIP image %s: %w", fileName, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close ZIP image %s: %w", fileName, closeErr)
		}

		if int64(len(data)) > MaxUploadSizeBytes {
			return nil, fmt.Errorf("ZIP image %s is too large, maximum allowed size per image is %d MB", fileName, MaxUploadSizeBytes/(1<<20))
		}

		contentType := mimetype.Detect(data).String()
		if !allowedImageContentTypes[contentType] {
			return nil, fmt.Errorf("ZIP contains unsupported file %s (%s). Only JPG, PNG, and WEBP images are allowed in ZIP uploads", fileName, contentType)
		}

		files = append(files, ValidatedFile{
			OriginalFileName: fileName,
			ContentType:      contentType,
			SizeBytes:        int64(len(data)),
			Data:             data,
		})
	}

	if len(files) == 0 {
		return nil, errors.New("ZIP archive does not contain any supported images")
	}

	return files, nil
}

func Reader(data []byte) io.Reader {
	return bytes.NewReader(data)
}

func shouldSkipZipEntry(originalName string, sanitizedName string) bool {
	normalizedName := strings.ReplaceAll(originalName, "\\", "/")
	if strings.HasPrefix(normalizedName, "__MACOSX/") {
		return true
	}

	return strings.HasPrefix(sanitizedName, ".")
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "/", "")

	return name
}
