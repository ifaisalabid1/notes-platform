package uploads

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"testing"
)

func TestValidateZipImageFilesAcceptsImages(t *testing.T) {
	zipData := buildTestZip(t, map[string][]byte{
		"gallery/first.png": tinyPNG(t),
		"second.png":        tinyPNG(t),
	})

	files, err := ValidateZipImageFiles(ValidatedFile{
		OriginalFileName: "gallery.zip",
		ContentType:      "application/zip",
		SizeBytes:        int64(len(zipData)),
		Data:             zipData,
		IsZIP:            true,
	})
	if err != nil {
		t.Fatalf("ValidateZipImageFiles returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 image files, got %d", len(files))
	}

	if files[0].OriginalFileName != "first.png" {
		t.Fatalf("expected sanitized nested file name, got %q", files[0].OriginalFileName)
	}

	if files[0].ContentType != "image/png" {
		t.Fatalf("expected image/png, got %q", files[0].ContentType)
	}
}

func TestValidateZipImageFilesRejectsNonImages(t *testing.T) {
	zipData := buildTestZip(t, map[string][]byte{
		"readme.txt": []byte("not an image"),
	})

	_, err := ValidateZipImageFiles(ValidatedFile{
		OriginalFileName: "gallery.zip",
		ContentType:      "application/zip",
		SizeBytes:        int64(len(zipData)),
		Data:             zipData,
		IsZIP:            true,
	})
	if err == nil {
		t.Fatal("expected non-image ZIP entry to be rejected")
	}
}

func buildTestZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for name, data := range files {
		entryWriter, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}

		if _, err := entryWriter.Write(data); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	return buffer.Bytes()
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}

	return data
}
