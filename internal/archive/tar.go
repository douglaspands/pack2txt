package archive

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EntryInfo represents metadata for an archived file or directory.
type EntryInfo struct {
	Path    string      `json:"path"`
	Size    int64       `json:"size"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"modTime"`
	IsDir   bool        `json:"isDir"`
}

// CreateTar packs files from rootPath into an in-memory Solid TAR archive byte slice.
func CreateTar(rootPath string, filter Filter) ([]byte, []EntryInfo, error) {
	stat, err := os.Stat(rootPath)
	if err != nil {
		return nil, nil, fmt.Errorf("stat error on '%s': %w", rootPath, err)
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	var entries []EntryInfo

	if !stat.IsDir() {
		// Single file packing
		fileName := filepath.Base(rootPath)
		info := EntryInfo{
			Path:    fileName,
			Size:    stat.Size(),
			Mode:    stat.Mode(),
			ModTime: stat.ModTime(),
			IsDir:   false,
		}

		hdr, err := tar.FileInfoHeader(stat, "")
		if err != nil {
			return nil, nil, fmt.Errorf("tar header error for file '%s': %w", rootPath, err)
		}
		hdr.Name = filepath.ToSlash(fileName)

		if err := tw.WriteHeader(hdr); err != nil {
			return nil, nil, fmt.Errorf("tar write header error: %w", err)
		}

		f, err := os.Open(rootPath)
		if err != nil {
			return nil, nil, fmt.Errorf("open file error: %w", err)
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return nil, nil, fmt.Errorf("tar write body error: %w", err)
		}

		if err := tw.Close(); err != nil {
			return nil, nil, fmt.Errorf("tar close error: %w", err)
		}

		return buf.Bytes(), []EntryInfo{info}, nil
	}

	// Directory packing
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, nil, fmt.Errorf("abs path error on '%s': %w", rootPath, err)
	}

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if path == absRoot {
			return nil
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		isDir := d.IsDir()

		if filter != nil && filter.ShouldIgnore(relPath, isDir) {
			if isDir {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Ensure forward slashes in tar header
		slashRelPath := filepath.ToSlash(relPath)
		if isDir && !strings.HasSuffix(slashRelPath, "/") {
			slashRelPath += "/"
		}
		hdr.Name = slashRelPath

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("tar write header for '%s': %w", slashRelPath, err)
		}

		entry := EntryInfo{
			Path:    slashRelPath,
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			IsDir:   isDir,
		}
		entries = append(entries, entry)

		if !isDir {
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("open file '%s': %w", path, err)
			}
			_, copyErr := io.Copy(tw, f)
			_ = f.Close()
			if copyErr != nil {
				return fmt.Errorf("tar write content for '%s': %w", path, copyErr)
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("walk error: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, nil, fmt.Errorf("tar writer close: %w", err)
	}

	return buf.Bytes(), entries, nil
}

// ExtractTar extracts an in-memory TAR archive into destDir with strict security.
func ExtractTar(tarBytes []byte, destDir string, overwrite bool) ([]EntryInfo, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination dir '%s': %w", destDir, err)
	}

	tr := tar.NewReader(bytes.NewReader(tarBytes))
	var extracted []EntryInfo

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read error: %w", err)
		}

		cleanPath, err := SanitizeExtractPath(destDir, hdr.Name)
		if err != nil {
			return nil, err
		}

		isDir := hdr.Typeflag == tar.TypeDir || strings.HasSuffix(hdr.Name, "/")

		if isDir {
			dirMode := SanitizeFileMode(hdr.FileInfo().Mode(), true)
			if err := os.MkdirAll(cleanPath, dirMode); err != nil {
				return nil, fmt.Errorf("mkdir '%s' failed: %w", cleanPath, err)
			}
		} else {
			// Ensure parent dir exists
			parent := filepath.Dir(cleanPath)
			if err := os.MkdirAll(parent, 0755); err != nil {
				return nil, fmt.Errorf("mkdir parent '%s' failed: %w", parent, err)
			}

			// Check overwrite
			if !overwrite {
				if _, err := os.Stat(cleanPath); err == nil {
					return nil, fmt.Errorf("destination file '%s' already exists (use --force to overwrite)", cleanPath)
				}
			}

			fileMode := SanitizeFileMode(hdr.FileInfo().Mode(), false)
			outFile, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode)
			if err != nil {
				return nil, fmt.Errorf("create file '%s' failed: %w", cleanPath, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return nil, fmt.Errorf("write file '%s' failed: %w", cleanPath, err)
			}
			_ = outFile.Close()

			// Restore modification time
			_ = os.Chtimes(cleanPath, time.Now(), hdr.ModTime)
		}

		extracted = append(extracted, EntryInfo{
			Path:    hdr.Name,
			Size:    hdr.Size,
			Mode:    hdr.FileInfo().Mode(),
			ModTime: hdr.ModTime,
			IsDir:   isDir,
		})
	}

	return extracted, nil
}

// ListTarEntries inspects an in-memory TAR archive without writing anything to disk.
func ListTarEntries(tarBytes []byte) ([]EntryInfo, error) {
	tr := tar.NewReader(bytes.NewReader(tarBytes))
	var entries []EntryInfo

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read error: %w", err)
		}

		isDir := hdr.Typeflag == tar.TypeDir || strings.HasSuffix(hdr.Name, "/")
		entries = append(entries, EntryInfo{
			Path:    hdr.Name,
			Size:    hdr.Size,
			Mode:    hdr.FileInfo().Mode(),
			ModTime: hdr.ModTime,
			IsDir:   isDir,
		})
	}

	return entries, nil
}
