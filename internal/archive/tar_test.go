package archive_test

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/douglas/pack2txt/internal/archive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchive_CreateAndExtractTar(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pack2txt_archive_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test folder hierarchy
	srcDir := filepath.Join(tmpDir, "source")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "subpkg", "nested"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "node_modules", "pkg"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, ".git"), 0755))

	file1 := filepath.Join(srcDir, "main.go")
	file2 := filepath.Join(srcDir, "subpkg", "helper.go")
	file3 := filepath.Join(srcDir, "subpkg", "nested", "deep.txt")
	ignoredFile := filepath.Join(srcDir, "node_modules", "pkg", "index.js")
	gitFile := filepath.Join(srcDir, ".git", "config")

	require.NoError(t, os.WriteFile(file1, []byte("package main\nfunc main(){}"), 0644))
	require.NoError(t, os.WriteFile(file2, []byte("package subpkg\nconst Version = 1"), 0644))
	require.NoError(t, os.WriteFile(file3, []byte("deep nested content"), 0644))
	require.NoError(t, os.WriteFile(ignoredFile, []byte("console.log('ignored')"), 0644))
	require.NoError(t, os.WriteFile(gitFile, []byte("[core] bare=false"), 0644))

	filter := archive.NewDefaultFilter(false)
	tarBytes, entries, err := archive.CreateTar(srcDir, filter)
	require.NoError(t, err)
	assert.NotEmpty(t, tarBytes)

	// Check that node_modules and .git were ignored
	for _, entry := range entries {
		assert.NotContains(t, entry.Path, "node_modules")
		assert.NotContains(t, entry.Path, ".git")
	}

	// Test In-memory inspection
	listed, err := archive.ListTarEntries(tarBytes)
	require.NoError(t, err)
	assert.Equal(t, len(entries), len(listed))

	// Test Extraction
	destDir := filepath.Join(tmpDir, "extracted")
	extracted, err := archive.ExtractTar(tarBytes, destDir, true)
	require.NoError(t, err)
	assert.Equal(t, len(entries), len(extracted))

	// Verify extracted files exist and content matches
	restored1, err := os.ReadFile(filepath.Join(destDir, "main.go"))
	require.NoError(t, err)
	assert.Equal(t, "package main\nfunc main(){}", string(restored1))

	restored3, err := os.ReadFile(filepath.Join(destDir, "subpkg", "nested", "deep.txt"))
	require.NoError(t, err)
	assert.Equal(t, "deep nested content", string(restored3))

	// Verify ignored files were not extracted
	_, err = os.Stat(filepath.Join(destDir, "node_modules"))
	assert.True(t, os.IsNotExist(err))
}

func TestArchive_ZipSlipProtection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pack2txt_zipslip_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Craft a malicious in-memory TAR with path traversal
	var maliciousBuf bytes.Buffer
	tw := tar.NewWriter(&maliciousBuf)

	maliciousEntries := []string{
		"../../evil.txt",
		"/etc/passwd",
		"sub/../../../etc/hosts",
		"..\\..\\windows_evil.bat",
	}

	for _, entryName := range maliciousEntries {
		hdr := &tar.Header{
			Name: entryName,
			Mode: 0644,
			Size: int64(len("malicious payload")),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, requireErr := tw.Write([]byte("malicious payload"))
		require.NoError(t, requireErr)
	}
	require.NoError(t, tw.Close())

	destDir := filepath.Join(tmpDir, "safe_dest")

	// Attempt extraction: must fail with ZipSlip error
	_, err = archive.ExtractTar(maliciousBuf.Bytes(), destDir, true)
	assert.Error(t, err)
	assert.ErrorIs(t, err, archive.ErrZipSlipViolation)

	// Ensure no evil file escaped to tmpDir
	_, err = os.Stat(filepath.Join(tmpDir, "evil.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestArchive_SingleFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pack2txt_single_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "single.txt")
	content := "Single standalone file packing."
	require.NoError(t, os.WriteFile(srcFile, []byte(content), 0644))

	tarBytes, entries, err := archive.CreateTar(srcFile, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "single.txt", entries[0].Path)

	destDir := filepath.Join(tmpDir, "dest_single")
	_, err = archive.ExtractTar(tarBytes, destDir, true)
	require.NoError(t, err)

	extractedContent, err := os.ReadFile(filepath.Join(destDir, "single.txt"))
	require.NoError(t, err)
	assert.Equal(t, content, string(extractedContent))
}
