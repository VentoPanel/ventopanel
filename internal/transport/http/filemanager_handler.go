package http

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/your-org/ventopanel/internal/filemanager"
)

type FileManagerHandler struct {
	svc *filemanager.Service
}

func NewFileManagerHandler(svc *filemanager.Service) *FileManagerHandler {
	return &FileManagerHandler{svc: svc}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func fmPath(c *gin.Context) string {
	p := c.Query("path")
	if p == "" {
		p = "/"
	}
	return p
}

func fmErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, filemanager.ErrForbidden):
		c.JSON(http.StatusForbidden, errorResponse{Error: err.Error()})
	case errors.Is(err, filemanager.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, filemanager.ErrIsDir):
		c.JSON(http.StatusBadRequest, errorResponse{Error: "path is a directory"})
	default:
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
	}
}

// ── Basic CRUD ────────────────────────────────────────────────────────────────

// ListDir GET /files?path=...
func (h *FileManagerHandler) ListDir(c *gin.Context) {
	items, err := h.svc.ListDir(fmPath(c))
	if err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "root": h.svc.RootPath})
}

// ReadFile GET /files/content?path=...
func (h *FileManagerHandler) ReadFile(c *gin.Context) {
	data, err := h.svc.ReadFile(fmPath(c))
	if err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": string(data), "path": fmPath(c)})
}

// WriteFile PUT /files/content?path=...
func (h *FileManagerHandler) WriteFile(c *gin.Context) {
	var body struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.WriteFile(fmPath(c), []byte(body.Content)); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// DeletePath DELETE /files?path=...
func (h *FileManagerHandler) DeletePath(c *gin.Context) {
	if err := h.svc.Delete(fmPath(c)); err != nil {
		fmErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateDir POST /files/dir?path=...
func (h *FileManagerHandler) CreateDir(c *gin.Context) {
	if err := h.svc.CreateDir(fmPath(c)); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}

// Rename POST /files/rename
func (h *FileManagerHandler) Rename(c *gin.Context) {
	var body struct {
		OldPath string `json:"old_path" binding:"required"`
		NewPath string `json:"new_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.Rename(body.OldPath, body.NewPath); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "renamed"})
}

// ── Upload ────────────────────────────────────────────────────────────────────

// Upload POST /files/upload?path=<directory>
func (h *FileManagerHandler) Upload(c *gin.Context) {
	dir := fmPath(c)
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "no files provided"})
		return
	}
	uploaded := make([]string, 0, len(files))
	for _, fh := range files {
		destPath := filepath.ToSlash(filepath.Join(dir, fh.Filename))
		src, openErr := fh.Open()
		if openErr != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: openErr.Error()})
			return
		}
		_, writeErr := h.svc.Upload(destPath, src)
		src.Close()
		if writeErr != nil {
			fmErr(c, writeErr)
			return
		}
		uploaded = append(uploaded, destPath)
	}
	c.JSON(http.StatusOK, gin.H{"uploaded": uploaded})
}

// ── Smart Download ────────────────────────────────────────────────────────────

// Download GET /files/download?path=...
//
// Smart: if path is a regular file — serves it directly with correct MIME type.
// If path is a directory — streams it as a ZIP archive on-the-fly via io.Pipe,
// without creating any temporary file on disk.
func (h *FileManagerHandler) Download(c *gin.Context) {
	p := fmPath(c)

	isDir, err := h.svc.IsDir(p)
	if err != nil {
		fmErr(c, err)
		return
	}

	if isDir {
		// Stream directory as ZIP via io.Pipe — no temp file.
		name := filepath.Base(p) + ".zip"
		c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
		c.Header("Content-Type", "application/zip")
		c.Header("Transfer-Encoding", "chunked")
		c.Status(http.StatusOK)

		pr, pw := io.Pipe()

		// Writer goroutine: streams zip into the pipe writer.
		go func() {
			err := h.svc.StreamDirAsZip(p, pw)
			pw.CloseWithError(err) // signals EOF or error to reader
		}()

		// Reader side is copied directly to the HTTP response.
		io.Copy(c.Writer, pr) //nolint:errcheck
		pr.Close()
		return
	}

	// Regular file download.
	f, size, err := h.svc.Download(p)
	if err != nil {
		fmErr(c, err)
		return
	}
	defer f.Close()

	name := filepath.Base(p)
	mimeType := mime.TypeByExtension(filepath.Ext(name))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Header("Content-Type", mimeType)
	if size > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", size))
	}
	c.Status(http.StatusOK)
	io.Copy(c.Writer, f) //nolint:errcheck
}

// ── Compress ──────────────────────────────────────────────────────────────────

// Compress POST /files/compress
// Body: { "src_paths": ["/dir1", "/file.txt"], "dest_zip": "/archive.zip" }
func (h *FileManagerHandler) Compress(c *gin.Context) {
	var body struct {
		SrcPaths []string `json:"src_paths" binding:"required,min=1"`
		DestZip  string   `json:"dest_zip"  binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.Compress(body.SrcPaths, body.DestZip); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "compressed", "dest": body.DestZip})
}

// ── Extract ───────────────────────────────────────────────────────────────────

// Extract POST /files/extract
// Body: { "zip_path": "/archive.zip", "dest_dir": "/output" }
func (h *FileManagerHandler) Extract(c *gin.Context) {
	var body struct {
		ZipPath string `json:"zip_path" binding:"required"`
		DestDir string `json:"dest_dir" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.Extract(body.ZipPath, body.DestDir); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "extracted", "dest": body.DestDir})
}

// ── Permissions ───────────────────────────────────────────────────────────────

// SetPermissions PATCH /files/permissions?path=...
// Body: { "mode": "755" }
func (h *FileManagerHandler) SetPermissions(c *gin.Context) {
	var body struct {
		Mode string `json:"mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.SetPermissions(fmPath(c), body.Mode); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "permissions updated", "mode": body.Mode})
}
