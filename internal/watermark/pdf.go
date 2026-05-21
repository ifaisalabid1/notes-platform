package watermark

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

type PDFWatermarker struct {
	text string
}

func NewPDFWatermarker(text string) (*PDFWatermarker, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("watermark text is required")
	}

	return &PDFWatermarker{
		text: text,
	}, nil
}

type WatermarkedPDF struct {
	Data []byte
}

func (w *PDFWatermarker) Apply(data []byte) (WatermarkedPDF, error) {
	if len(data) == 0 {
		return WatermarkedPDF{}, fmt.Errorf("pdf data is required")
	}

	tempDir, err := os.MkdirTemp("", "notes-platform-pdf-*")
	if err != nil {
		return WatermarkedPDF{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, uuid.NewString()+"-input.pdf")
	outputPath := filepath.Join(tempDir, uuid.NewString()+"-watermarked.pdf")

	if err := os.WriteFile(inputPath, data, 0o600); err != nil {
		return WatermarkedPDF{}, fmt.Errorf("write temp input pdf: %w", err)
	}

	description := strings.Join([]string{
		"font:Helvetica",
		"points:42",
		"scale:.8 rel",
		"diagonal:1",
		"opacity:.18",
		"color:.45 .45 .45",
	}, ", ")

	onTop := false

	err = api.AddTextWatermarksFile(
		inputPath,
		outputPath,
		nil,
		onTop,
		w.text,
		description,
		nil,
	)
	if err != nil {
		return WatermarkedPDF{}, fmt.Errorf("apply pdf watermark: %w", err)
	}

	watermarkedData, err := os.ReadFile(outputPath)
	if err != nil {
		return WatermarkedPDF{}, fmt.Errorf("read watermarked pdf: %w", err)
	}

	if len(watermarkedData) == 0 {
		return WatermarkedPDF{}, fmt.Errorf("watermarked pdf is empty")
	}

	return WatermarkedPDF{
		Data: watermarkedData,
	}, nil
}
