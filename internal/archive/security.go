package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrZipSlipViolation is returned when an archive entry attempts to extract outside destination.
	ErrZipSlipViolation = errors.New("security violation: path traversal / zip slip detected")
)

// SanitizeExtractPath checks that the target file path stays strictly inside destDir.
func SanitizeExtractPath(destDir, entryPath string) (string, error) {
	// Normalize separators to OS native
	cleanEntry := filepath.Clean(entryPath)

	// Disallow drive letters (Windows) or absolute paths
	if filepath.IsAbs(cleanEntry) || strings.HasPrefix(cleanEntry, "/") || strings.HasPrefix(cleanEntry, "\\") {
		// Strip leading volume / root to make it relative
		cleanEntry = strings.TrimLeft(cleanEntry, "/\\")
		if vol := filepath.VolumeName(cleanEntry); vol != "" {
			cleanEntry = strings.TrimPrefix(cleanEntry, vol)
			cleanEntry = strings.TrimLeft(cleanEntry, "/\\")
		}
	}

	cleanDest, err := filepath.Abs(destDir)
	if err != nil {
		return "", fmt.Errorf("invalid destination directory: %w", err)
	}

	targetPath := filepath.Join(cleanDest, cleanEntry)
	targetPath = filepath.Clean(targetPath)

	// Ensure targetPath is inside cleanDest
	rel, err := filepath.Rel(cleanDest, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." && cleanEntry != "." && cleanEntry != "" {
		return "", fmt.Errorf("%w: entry '%s' escapes destination '%s'", ErrZipSlipViolation, entryPath, destDir)
	}

	return targetPath, nil
}

// SanitizeFileMode masks dangerous execution/permission bits.
func SanitizeFileMode(mode os.FileMode, isDir bool) os.FileMode {
	perm := mode.Perm() & 0777
	if isDir {
		return perm | 0755
	}
	// Ensure at least read permission
	return perm | 0644
}
