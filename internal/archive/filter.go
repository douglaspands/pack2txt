package archive

import (
	"path/filepath"
	"strings"
)

// Filter defines an interface for deciding if a file or directory should be ignored.
type Filter interface {
	ShouldIgnore(relPath string, isDir bool) bool
}

// DefaultIgnoredDirs are common dependency and cache directories ignored by default.
var DefaultIgnoredDirs = map[string]bool{
	".git":         true,
	".github":      false, // allow github workflows if needed
	".venv":        true,
	"venv":         true,
	"node_modules": true,
	"__pycache__":  true,
	".pytest_cache": true,
	".mypy_cache":  true,
	".DS_Store":    true,
	"dist":         true,
	"build":        true,
	"out":          true,
	"target":       true,
	".next":        true,
	".turbo":       true,
	".idea":        true,
	".vscode":      true,
	"bin":          true,
	"obj":          true,
}

// DefaultIgnoredFiles are common temporary or compiled file names/extensions.
var DefaultIgnoredFiles = []string{
	".DS_Store",
	"Thumbs.db",
	"*.pyc",
	"*.pyo",
	"*.pyd",
	"*.o",
	"*.a",
	"*.so",
	"*.dylib",
	"*.exe",
}

// DefaultFilter implements standard dev-file exclusion rules.
type DefaultFilter struct {
	Disabled bool
}

// NewDefaultFilter returns a filter with standard developer ignore rules.
func NewDefaultFilter(noIgnore bool) *DefaultFilter {
	return &DefaultFilter{
		Disabled: noIgnore,
	}
}

func (f *DefaultFilter) ShouldIgnore(relPath string, isDir bool) bool {
	if f.Disabled {
		return false
	}

	normPath := filepath.ToSlash(relPath)
	parts := strings.Split(normPath, "/")

	for _, part := range parts {
		if part == "" {
			continue
		}

		if isDir || part != parts[len(parts)-1] {
			if DefaultIgnoredDirs[part] {
				return true
			}
		}

		if part == ".DS_Store" || part == "Thumbs.db" {
			return true
		}

		if strings.HasPrefix(part, ".git") && part == ".git" {
			return true
		}
	}

	// File extension checks
	fileName := parts[len(parts)-1]
	for _, pattern := range DefaultIgnoredFiles {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
	}

	return false
}
