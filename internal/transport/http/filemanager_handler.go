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
	factory *filemanager.Factory
}

func NewFileManagerHandler(factory *filemanager.Factory) *FileManagerHandler {
	return &FileManagerHandler{factory: factory}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// fmSvc resolves the correct *filemanager.Service for this request.
// If the "server_id" query param is present the handler uses a remote SFTP
// service; otherwise it falls back to the local filesystem service.
func (h *FileManagerHandler) fmSvc(c *gin.Context) (*filemanager.Service, error) {
	return h.factory.Resolve(c.Request.Context(), c.Query("server_id"))
}

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

// ListDir GET /files?path=...&server_id=...
func (h *FileManagerHandler) ListDir(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	items, err := svc.ListDir(fmPath(c))
	if err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "root": svc.RootPath, "server_id": c.Query("server_id")})
}

// ReadFile GET /files/content?path=...&server_id=...
func (h *FileManagerHandler) ReadFile(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	data, err := svc.ReadFile(fmPath(c))
	if err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": string(data), "path": fmPath(c)})
}

// WriteFile PUT /files/content?path=...&server_id=...
func (h *FileManagerHandler) WriteFile(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := svc.WriteFile(fmPath(c), []byte(body.Content)); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// DeletePath DELETE /files?path=...&server_id=...
func (h *FileManagerHandler) DeletePath(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	if err := svc.Delete(fmPath(c)); err != nil {
		fmErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateDir POST /files/dir?path=...&server_id=...
func (h *FileManagerHandler) CreateDir(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	if err := svc.CreateDir(fmPath(c)); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}

// Rename POST /files/rename?server_id=...
func (h *FileManagerHandler) Rename(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	var body struct {
		OldPath string `json:"old_path" binding:"required"`
		NewPath string `json:"new_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := svc.Rename(body.OldPath, body.NewPath); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "renamed"})
}

// ── Upload ────────────────────────────────────────────────────────────────────

// Upload POST /files/upload?path=<directory>&server_id=...
func (h *FileManagerHandler) Upload(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
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
		_, writeErr := svc.Upload(destPath, src)
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

// Download GET /files/download?path=...&server_id=...
func (h *FileManagerHandler) Download(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	p := fmPath(c)

	isDir, err := svc.IsDir(p)
	if err != nil {
		fmErr(c, err)
		return
	}

	if isDir {
		name := filepath.Base(p) + ".zip"
		c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
		c.Header("Content-Type", "application/zip")
		c.Header("Transfer-Encoding", "chunked")
		c.Status(http.StatusOK)

		pr, pw := io.Pipe()
		go func() {
			pw.CloseWithError(svc.StreamDirAsZip(p, pw))
		}()
		io.Copy(c.Writer, pr) //nolint:errcheck
		pr.Close()
		return
	}

	f, size, err := svc.Download(p)
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

// Compress POST /files/compress?server_id=...
// Body: { "src_paths": ["/dir1", "/file.txt"], "dest_zip": "/archive.zip" }
func (h *FileManagerHandler) Compress(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	var body struct {
		SrcPaths []string `json:"src_paths" binding:"required,min=1"`
		DestZip  string   `json:"dest_zip"  binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := svc.Compress(body.SrcPaths, body.DestZip); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "compressed", "dest": body.DestZip})
}

// ── Extract ───────────────────────────────────────────────────────────────────

// Extract POST /files/extract?server_id=...
// Body: { "zip_path": "/archive.zip", "dest_dir": "/output" }
func (h *FileManagerHandler) Extract(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	var body struct {
		ZipPath string `json:"zip_path" binding:"required"`
		DestDir string `json:"dest_dir" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := svc.Extract(body.ZipPath, body.DestDir); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "extracted", "dest": body.DestDir})
}

// ── Permissions ───────────────────────────────────────────────────────────────

// SetPermissions PATCH /files/permissions?path=...&server_id=...
// Body: { "mode": "755" }
func (h *FileManagerHandler) SetPermissions(c *gin.Context) {
	svc, err := h.fmSvc(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	var body struct {
		Mode string `json:"mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := svc.SetPermissions(fmPath(c), body.Mode); err != nil {
		fmErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "permissions updated", "mode": body.Mode})
}
