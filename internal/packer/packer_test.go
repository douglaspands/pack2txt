package packer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/douglas/pack2txt/internal/compressor"
	"github.com/douglas/pack2txt/internal/encoder"
	"github.com/douglas/pack2txt/internal/packer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPacker_EndToEnd_AllEncodersAndCompressors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pack2txt_e2e_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Build sample tree
	srcDir := filepath.Join(tmpDir, "source_repo")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "cmd", "app"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "pkg", "utils"), 0755))

	mainGo := "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"Hello pack2txt!\") }\n"
	utilsGo := "package utils\nconst Version = \"1.0.0\"\nfunc Helper() string { return \"helper\" }\n"
	readmeMd := "# Sample Project\nThis is a test project to verify pack2txt end-to-end.\n"

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "cmd", "app", "main.go"), []byte(mainGo), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "pkg", "utils", "utils.go"), []byte(utilsGo), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "README.md"), []byte(readmeMd), 0644))

	testCombos := []struct {
		comp string
		enc  string
	}{
		{compressor.NameBrotli, encoder.NameBase32768},
		{compressor.NameBrotli, encoder.NameBase91},
		{compressor.NameZstd, encoder.NameBase85},
		{compressor.NameGzip, encoder.NameBase64},
		{compressor.NameAuto, encoder.NameBase32768},
		{compressor.NameNone, encoder.NameBase64},
	}

	for _, tc := range testCombos {
		comboName := tc.comp + "_" + tc.enc
		t.Run(comboName, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, comboName+".txt")

			// 1. Pack
			packRes, err := packer.Pack(packer.PackOptions{
				SourcePath: srcDir,
				Compressor: tc.comp,
				Encoder:    tc.enc,
				OutputPath: outFile,
			})
			require.NoError(t, err)
			assert.NotEmpty(t, packRes.Envelope)
			assert.True(t, strings.HasPrefix(packRes.Envelope, "PACK2TXT:v1:"))
			assert.Equal(t, 3, packRes.FileCount)
			assert.FileExists(t, outFile)

			// 2. Inspect
			inspectRes, err := packer.Inspect(packer.InspectOptions{
				InputPath: outFile,
			})
			require.NoError(t, err)
			assert.Equal(t, 3, inspectRes.FileCount)
			assert.Equal(t, packRes.UncompressedSize, inspectRes.UncompressedSize)

			// 3. Unpack
			destDir := filepath.Join(tmpDir, "extracted_"+comboName)
			unpackRes, err := packer.Unpack(packer.UnpackOptions{
				InputPath: outFile,
				DestDir:   destDir,
				Overwrite: true,
			})
			require.NoError(t, err)
			assert.Equal(t, 3, unpackRes.FileCount)

			// 4. Verify exact file restoration
			restoredMain, err := os.ReadFile(filepath.Join(destDir, "cmd", "app", "main.go"))
			require.NoError(t, err)
			assert.Equal(t, mainGo, string(restoredMain))

			restoredUtils, err := os.ReadFile(filepath.Join(destDir, "pkg", "utils", "utils.go"))
			require.NoError(t, err)
			assert.Equal(t, utilsGo, string(restoredUtils))

			restoredReadme, err := os.ReadFile(filepath.Join(destDir, "README.md"))
			require.NoError(t, err)
			assert.Equal(t, readmeMd, string(restoredReadme))
		})
	}
}

func TestPacker_FallbackDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pack2txt_fallback_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "input.txt")
	testData := "This is raw fallback testing without the PACK2TXT envelope header."
	require.NoError(t, os.WriteFile(srcFile, []byte(testData), 0644))

	// Pack with standard envelope
	packRes, err := packer.Pack(packer.PackOptions{
		SourcePath: srcFile,
		Compressor: compressor.NameGzip,
		Encoder:    encoder.NameBase64,
	})
	require.NoError(t, err)

	// Strip the "PACK2TXT:v1:gzip:b64:" prefix to simulate raw payload
	parts := strings.SplitN(packRes.Envelope, ":", 5)
	rawPayload := parts[4]

	// Unpack raw payload directly
	destDir := filepath.Join(tmpDir, "extracted_raw")
	unpackRes, err := packer.Unpack(packer.UnpackOptions{
		RawInput:  rawPayload,
		DestDir:   destDir,
		Overwrite: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, unpackRes.FileCount)

	restored, err := os.ReadFile(filepath.Join(destDir, "input.txt"))
	require.NoError(t, err)
	assert.Equal(t, testData, string(restored))
}
