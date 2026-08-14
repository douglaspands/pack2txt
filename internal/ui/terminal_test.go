package ui_test

import (
	"testing"
	"time"

	"github.com/douglas/pack2txt/internal/packer"
	"github.com/douglas/pack2txt/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "500 B", ui.FormatBytes(500))
	assert.Equal(t, "1.00 KB", ui.FormatBytes(1024))
	assert.Equal(t, "1.50 KB", ui.FormatBytes(1536))
	assert.Equal(t, "1.00 MB", ui.FormatBytes(1024*1024))
	assert.Equal(t, "2.50 GB", ui.FormatBytes(int64(2.5*1024*1024*1024)))
}

func TestRenderFunctions_DoNotPanic(t *testing.T) {
	packRes := &packer.PackResult{
		Envelope:         "PACK2TXT:v1:brotli:b32768:test",
		FileCount:        5,
		UncompressedSize: 10240,
		CompressedSize:   2048,
		EncodedChars:     1100,
		SavingsPercent:   80.0,
		Compressor:       "brotli",
		Encoder:          "b32768",
		Duration:         15 * time.Millisecond,
		OutputPath:       "out.txt",
	}

	assert.NotPanics(t, func() {
		ui.RenderPackResult(packRes, true)  // pipe mode
		ui.RenderPackResult(packRes, false) // normal
	})

	inspectRes := &packer.InspectResult{
		FileCount:        5,
		DirCount:         2,
		UncompressedSize: 10240,
		CompressedSize:   2048,
		EncodedChars:     1100,
		SavingsPercent:   80.0,
		Compressor:       "brotli",
		Encoder:          "b32768",
	}

	assert.NotPanics(t, func() {
		ui.RenderInspectResult(inspectRes)
	})

	unpackRes := &packer.UnpackResult{
		FileCount:  5,
		TotalSize:  10240,
		Compressor: "brotli",
		Encoder:    "b32768",
		DestDir:    "/tmp/test",
		Duration:   10 * time.Millisecond,
	}

	assert.NotPanics(t, func() {
		ui.RenderUnpackResult(unpackRes, true)  // quiet
		ui.RenderUnpackResult(unpackRes, false) // normal
	})
}
