package filemanager

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/afero"
)

// Service provides jailed filesystem operations backed by an afero.Fs.
// All paths supplied by callers are relative to RootPath; any attempt to
// escape the root via ".." or absolute paths is rejected with ErrForbidden.
type Service struct {
	fs       afero.Fs
	RootPath string // absolute path on the host OS
}

// NewService creates a new Service.
// rootPath is the absolute directory that all operations are restricted to.
func NewService(rootPath string) *Service {
	return &Service{
		fs:       afero.NewBasePathFs(afero.NewOsFs(), rootPath),
		RootPath: rootPath,
	}
}

// safePath validates that a caller-supplied relative path does not escape the
// root after cleaning. It returns the cleaned relative path or ErrForbidden.
func safePath(rel string) (string, error) {
	rel = filepath.ToSlash(filepath.Clean("/" + rel))
	if strings.Contains(rel, "..") {
		return "", ErrForbidden
	}
	return rel, nil
}

// ── Directory listing ────────────────────────────────────────────────────────

// ListDir returns the contents of the directory at relPath.
func (s *Service) ListDir(relPath string) ([]FileItem, error) {
	clean, err := safePath(relPath)
	if err != nil {
		return nil, err
	}
	entries, err := afero.ReadDir(s.fs, clean)
	if err != nil {
		return nil, mapErr(err)
	}
	items := make([]FileItem, 0, len(entries))
	for _, e := range entries {
		itemPath := filepath.ToSlash(filepath.Join(clean, e.Name()))
		ext := ""
		if !e.IsDir() {
			ext = strings.ToLower(filepath.Ext(e.Name()))
		}
		items = append(items, FileItem{
			Name:    e.Name(),
			Path:    itemPath,
			IsDir:   e.IsDir(),
			Size:    e.Size(),
			ModTime: e.ModTime(),
			Ext:     ext,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

// ── Read / Write ─────────────────────────────────────────────────────────────

// ReadFile returns the contents of the file at relPath.
func (s *Service) ReadFile(relPath string) ([]byte, error) {
	clean, err := safePath(relPath)
	if err != nil {
		return nil, err
	}
	data, err := afero.ReadFile(s.fs, clean)
	return data, mapErr(err)
}

// WriteFile writes content to the file at relPath, creating it if necessary.
func (s *Service) WriteFile(relPath string, content []byte) error {
	clean, err := safePath(relPath)
	if err != nil {
		return err
	}
	if mkErr := s.fs.MkdirAll(filepath.Dir(clean), 0755); mkErr != nil {
		return mkErr
	}
	return afero.WriteFile(s.fs, clean, content, 0644)
}

// ── CRUD ─────────────────────────────────────────────────────────────────────

// Delete removes a file or directory (recursively) at relPath.
func (s *Service) Delete(relPath string) error {
	clean, err := safePath(relPath)
	if err != nil {
		return err
	}
	if clean == "/" {
		return ErrForbidden
	}
	return mapErr(s.fs.RemoveAll(clean))
}

// CreateDir creates the directory tree at relPath.
func (s *Service) CreateDir(relPath string) error {
	clean, err := safePath(relPath)
	if err != nil {
		return err
	}
	return mapErr(s.fs.MkdirAll(clean, 0755))
}

// Rename moves/renames oldPath to newPath (both relative).
func (s *Service) Rename(oldPath, newPath string) error {
	cleanOld, err := safePath(oldPath)
	if err != nil {
		return err
	}
	cleanNew, err := safePath(newPath)
	if err != nil {
		return err
	}
	return mapErr(s.fs.Rename(cleanOld, cleanNew))
}

// ── Permissions ──────────────────────────────────────────────────────────────

// SetPermissions changes the mode of the file/dir at relPath.
// mode is a Unix permission string like "755" or "644".
func (s *Service) SetPermissions(relPath string, mode string) error {
	clean, err := safePath(relPath)
	if err != nil {
		return err
	}
	perm, err := strconv.ParseUint(strings.TrimSpace(mode), 8, 32)
	if err != nil {
		return fmt.Errorf("invalid mode %q: must be an octal string like \"755\"", mode)
	}
	return mapErr(s.fs.Chmod(clean, os.FileMode(perm)))
}

// ── Upload / Download ────────────────────────────────────────────────────────

// Upload streams reader into the file at relPath using io.Copy.
func (s *Service) Upload(relPath string, r io.Reader) (int64, error) {
	clean, err := safePath(relPath)
	if err != nil {
		return 0, err
	}
	if mkErr := s.fs.MkdirAll(filepath.Dir(clean), 0755); mkErr != nil {
		return 0, mkErr
	}
	f, err := s.fs.Create(clean)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)
}

// IsDir reports whether relPath is a directory.
func (s *Service) IsDir(relPath string) (bool, error) {
	clean, err := safePath(relPath)
	if err != nil {
		return false, err
	}
	info, err := s.fs.Stat(clean)
	if err != nil {
		return false, mapErr(err)
	}
	return info.IsDir(), nil
}

// Download opens a regular file and returns a ReadCloser + size.
// For directories use StreamDirAsZip instead.
func (s *Service) Download(relPath string) (afero.File, int64, error) {
	clean, err := safePath(relPath)
	if err != nil {
		return nil, 0, err
	}
	info, err := s.fs.Stat(clean)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	if info.IsDir() {
		return nil, 0, ErrIsDir
	}
	f, err := s.fs.Open(clean)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	return f, info.Size(), nil
}

// StreamDirAsZip writes the contents of the directory at relPath as a ZIP
// archive directly into w, without creating any temporary file on disk.
// It is safe to pipe w into an http.ResponseWriter.
func (s *Service) StreamDirAsZip(relPath string, w io.Writer) error {
	clean, err := safePath(relPath)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(w)
	defer zw.Close()

	return afero.Walk(s.fs, clean, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		// Entry name is relative to the directory being zipped.
		rel := strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(clean))
		rel = strings.TrimPrefix(rel, "/")

		header := &zip.FileHeader{
			Name:   rel,
			Method: zip.Deflate,
		}
		header.SetModTime(info.ModTime())

		fw, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		f, err := s.fs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(fw, f)
		return err
	})
}

// ── Archive: Compress ────────────────────────────────────────────────────────

// Compress creates a ZIP archive at destZip containing all paths in srcPaths.
// Each srcPath is validated with safePath. Directories are added recursively.
func (s *Service) Compress(srcPaths []string, destZip string) error {
	cleanDest, err := safePath(destZip)
	if err != nil {
		return err
	}
	if mkErr := s.fs.MkdirAll(filepath.Dir(cleanDest), 0755); mkErr != nil {
		return mkErr
	}

	out, err := s.fs.Create(cleanDest)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, src := range srcPaths {
		cleanSrc, err := safePath(src)
		if err != nil {
			return err
		}
		if err := s.addToZip(zw, cleanSrc, filepath.Base(cleanSrc)); err != nil {
			return err
		}
	}
	return nil
}

// addToZip recursively adds path (and its children if it is a directory)
// to zw. prefix is the path that will appear inside the archive.
func (s *Service) addToZip(zw *zip.Writer, fsPath, prefix string) error {
	info, err := s.fs.Stat(fsPath)
	if err != nil {
		return mapErr(err)
	}

	if !info.IsDir() {
		return s.addFileToZip(zw, fsPath, prefix)
	}

	// Directory: walk recursively.
	return afero.Walk(s.fs, fsPath, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil || fi.IsDir() {
			return walkErr
		}
		// Make entry name relative to the parent of fsPath.
		rel := strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(filepath.Dir(fsPath))+"/")
		return s.addFileToZip(zw, path, rel)
	})
}

func (s *Service) addFileToZip(zw *zip.Writer, fsPath, entryName string) error {
	header := &zip.FileHeader{
		Name:   filepath.ToSlash(entryName),
		Method: zip.Deflate,
	}
	info, err := s.fs.Stat(fsPath)
	if err == nil {
		header.SetModTime(info.ModTime())
	}

	fw, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	f, err := s.fs.Open(fsPath)
	if err != nil {
		return mapErr(err)
	}
	defer f.Close()

	_, err = io.Copy(fw, f)
	return err
}

// ── Archive: Extract ─────────────────────────────────────────────────────────

// Extract unpacks the ZIP archive at zipPath into destDir.
// Both paths are validated with safePath. No files outside destDir are created.
func (s *Service) Extract(zipPath string, destDir string) error {
	cleanZip, err := safePath(zipPath)
	if err != nil {
		return err
	}
	cleanDest, err := safePath(destDir)
	if err != nil {
		return err
	}

	// Read the entire zip into memory so we can use zip.NewReader (needs ReaderAt).
	data, err := afero.ReadFile(s.fs, cleanZip)
	if err != nil {
		return mapErr(err)
	}

	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("invalid zip: %w", err)
	}

	for _, f := range r.File {
		// Sanitise entry name – prevent zip-slip.
		entryPath := filepath.ToSlash(filepath.Join(cleanDest, filepath.Clean("/"+f.Name)))
		if !strings.HasPrefix(entryPath, filepath.ToSlash(cleanDest)) {
			return ErrForbidden
		}

		if f.FileInfo().IsDir() {
			if mkErr := s.fs.MkdirAll(entryPath, 0755); mkErr != nil {
				return mkErr
			}
			continue
		}

		if mkErr := s.fs.MkdirAll(filepath.Dir(entryPath), 0755); mkErr != nil {
			return mkErr
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		dst, err := s.fs.Create(entryPath)
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(dst, rc)
		rc.Close()
		dst.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// mapErr converts afero/os errors to domain sentinel errors where applicable.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "no such file") || strings.Contains(msg, "not found") {
		return fmt.Errorf("%w: %s", ErrNotFound, err)
	}
	return err
}
