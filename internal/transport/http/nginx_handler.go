package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	gossh "golang.org/x/crypto/ssh"

	"github.com/your-org/ventopanel/internal/filemanager"
)

// nginxShellescape wraps s in single quotes, escaping any existing single quotes.
func nginxShellescape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// ─── Handler ──────────────────────────────────────────────────────────────────

type NginxHandler struct {
	servers serverGetter
}

func NewNginxHandler(servers serverGetter) *NginxHandler {
	return &NginxHandler{servers: servers}
}

// ─── Request / Response types ─────────────────────────────────────────────────

type NginxVhost struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}

type NginxStatusResponse struct {
	Active  bool   `json:"active"`
	Version string `json:"version"`
	Config  string `json:"config_test"` // "ok" or error text
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// nginxRun executes a command on the remote server and returns stdout+stderr combined.
func nginxRun(sshCli *gossh.Client, cmd string) string {
	sess, err := sshCli.NewSession()
	if err != nil {
		return ""
	}
	defer sess.Close()

	out, _ := sess.CombinedOutput(cmd)
	return strings.TrimSpace(string(out))
}

// getSSH resolves the server from DB, opens an SSH connection, and returns it.
func (h *NginxHandler) getSSH(c *gin.Context) (*gossh.Client, bool) {
	serverID := c.Param("id")
	srv, err := h.servers.GetByID(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "server not found"})
		return nil, false
	}
	dialCfg := filemanager.ServerDialConfig{
		Host:     srv.Host,
		Port:     srv.Port,
		User:     srv.SSHUser,
		Password: srv.SSHPassword,
	}
	_, sshCli, err := filemanager.GlobalPool.Get(serverID, dialCfg)
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: "SSH connection failed: " + err.Error()})
		return nil, false
	}
	return sshCli, true
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// ListVhosts godoc
// GET /servers/:id/nginx/vhosts
// Returns all files from /etc/nginx/sites-available with enabled status.
func (h *NginxHandler) ListVhosts(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}

	// List sites-available
	availableOut := nginxRun(sshCli, "ls /etc/nginx/sites-available 2>/dev/null")
	// List sites-enabled (symlinks)
	enabledOut := nginxRun(sshCli, "ls /etc/nginx/sites-enabled 2>/dev/null")

	enabledSet := map[string]bool{}
	for _, name := range strings.Fields(enabledOut) {
		enabledSet[name] = true
	}

	var vhosts []NginxVhost
	for _, name := range strings.Fields(availableOut) {
		vhosts = append(vhosts, NginxVhost{
			Name:    name,
			Enabled: enabledSet[name],
			Path:    "/etc/nginx/sites-available/" + name,
		})
	}
	if vhosts == nil {
		vhosts = []NginxVhost{}
	}

	c.JSON(http.StatusOK, vhosts)
}

// GetVhost godoc
// GET /servers/:id/nginx/vhosts/:name
// Returns the raw config file content.
func (h *NginxHandler) GetVhost(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	name := c.Param("name")
	content := nginxRun(sshCli, fmt.Sprintf("cat /etc/nginx/sites-available/%s 2>&1", nginxShellescape(name)))
	c.JSON(http.StatusOK, gin.H{"name": name, "content": content})
}

// SaveVhost godoc
// PUT /servers/:id/nginx/vhosts/:name
// Writes config, runs nginx -t, returns test result.
func (h *NginxHandler) SaveVhost(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	var body struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "content is required"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	name := c.Param("name")
	path := "/etc/nginx/sites-available/" + name

	// Write via tee to handle permissions (requires sudo-less root or sudoers).
	writeCmd := fmt.Sprintf("cat > %s", nginxShellescape(path))
	sess, err := sshCli.NewSession()
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: "SSH session failed"})
		return
	}
	sess.Stdin = strings.NewReader(body.Content)
	if err := sess.Run(writeCmd); err != nil {
		sess.Close()
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to write config: " + err.Error()})
		return
	}
	sess.Close()

	// Test nginx config.
	testOut := nginxRun(sshCli, "nginx -t 2>&1")
	testOK := strings.Contains(testOut, "test is successful") || strings.Contains(testOut, "syntax is ok")
	c.JSON(http.StatusOK, gin.H{"test_ok": testOK, "test_output": testOut})
}

// CreateVhost godoc
// POST /servers/:id/nginx/vhosts
// Creates a new vhost config file.
func (h *NginxHandler) CreateVhost(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	var body struct {
		Name    string `json:"name"    binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "name is required"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	path := "/etc/nginx/sites-available/" + body.Name
	content := body.Content
	if content == "" {
		content = defaultVhostTemplate(body.Name)
	}

	sess, err := sshCli.NewSession()
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: "SSH session failed"})
		return
	}
	sess.Stdin = strings.NewReader(content)
	if err := sess.Run(fmt.Sprintf("cat > %s", nginxShellescape(path))); err != nil {
		sess.Close()
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to create config: " + err.Error()})
		return
	}
	sess.Close()
	c.JSON(http.StatusCreated, gin.H{"name": body.Name, "path": path})
}

// DeleteVhost godoc
// DELETE /servers/:id/nginx/vhosts/:name
func (h *NginxHandler) DeleteVhost(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	name := c.Param("name")
	// Disable first (remove symlink), then delete the file.
	nginxRun(sshCli, fmt.Sprintf("rm -f /etc/nginx/sites-enabled/%s", nginxShellescape(name)))
	nginxRun(sshCli, fmt.Sprintf("rm -f /etc/nginx/sites-available/%s", nginxShellescape(name)))
	c.JSON(http.StatusOK, gin.H{"deleted": name})
}

// EnableVhost godoc
// POST /servers/:id/nginx/vhosts/:name/enable
func (h *NginxHandler) EnableVhost(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	name := c.Param("name")
	out := nginxRun(sshCli, fmt.Sprintf(
		"ln -sf /etc/nginx/sites-available/%s /etc/nginx/sites-enabled/%s 2>&1",
		nginxShellescape(name), nginxShellescape(name),
	))
	testOut := nginxRun(sshCli, "nginx -t 2>&1")
	testOK := strings.Contains(testOut, "test is successful") || strings.Contains(testOut, "syntax is ok")
	c.JSON(http.StatusOK, gin.H{"enabled": true, "ln_output": out, "test_ok": testOK, "test_output": testOut})
}

// DisableVhost godoc
// POST /servers/:id/nginx/vhosts/:name/disable
func (h *NginxHandler) DisableVhost(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	name := c.Param("name")
	out := nginxRun(sshCli, fmt.Sprintf("rm -f /etc/nginx/sites-enabled/%s 2>&1", nginxShellescape(name)))
	testOut := nginxRun(sshCli, "nginx -t 2>&1")
	testOK := strings.Contains(testOut, "test is successful") || strings.Contains(testOut, "syntax is ok")
	c.JSON(http.StatusOK, gin.H{"enabled": false, "rm_output": out, "test_ok": testOK, "test_output": testOut})
}

// TestConfig godoc
// POST /servers/:id/nginx/test
func (h *NginxHandler) TestConfig(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	out := nginxRun(sshCli, "nginx -t 2>&1")
	ok2 := strings.Contains(out, "test is successful") || strings.Contains(out, "syntax is ok")
	c.JSON(http.StatusOK, gin.H{"ok": ok2, "output": out})
}

// Reload godoc
// POST /servers/:id/nginx/reload
func (h *NginxHandler) Reload(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	// Try nginx -s reload, fall back to systemctl reload nginx.
	out := nginxRun(sshCli, "nginx -s reload 2>&1 || systemctl reload nginx 2>&1")
	c.JSON(http.StatusOK, gin.H{"output": out})
}

// Status godoc
// GET /servers/:id/nginx/status
func (h *NginxHandler) Status(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	activeOut := nginxRun(sshCli, "systemctl is-active nginx 2>/dev/null || service nginx status 2>/dev/null | grep -o 'running\\|stopped' | head -1")
	versionOut := nginxRun(sshCli, "nginx -v 2>&1 | head -1")
	testOut := nginxRun(sshCli, "nginx -t 2>&1")
	testOK := strings.Contains(testOut, "test is successful") || strings.Contains(testOut, "syntax is ok")
	configTest := "ok"
	if !testOK {
		configTest = testOut
	}
	c.JSON(http.StatusOK, NginxStatusResponse{
		Active:  strings.TrimSpace(activeOut) == "active" || strings.Contains(activeOut, "running"),
		Version: versionOut,
		Config:  configTest,
	})
}

// IssueCert godoc
// POST /servers/:id/nginx/vhosts/:name/ssl
// Runs certbot --nginx for the given domain.
func (h *NginxHandler) IssueCert(c *gin.Context) {
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	var body struct {
		Domain string `json:"domain" binding:"required"`
		Email  string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "domain is required"})
		return
	}
	sshCli, ok := h.getSSH(c)
	if !ok {
		return
	}
	emailFlag := ""
	if body.Email != "" {
		emailFlag = "--email " + nginxShellescape(body.Email) + " --no-eff-email"
	} else {
		emailFlag = "--register-unsafely-without-email"
	}
	cmd := fmt.Sprintf(
		"certbot --nginx -d %s %s --agree-tos --non-interactive 2>&1",
		nginxShellescape(body.Domain), emailFlag,
	)
	out := nginxRun(sshCli, cmd)
	success := strings.Contains(out, "Congratulations") || strings.Contains(out, "Certificate not yet due for renewal")
	c.JSON(http.StatusOK, gin.H{"success": success, "output": out})
}

// ─── Default template ─────────────────────────────────────────────────────────

func defaultVhostTemplate(name string) string {
	return fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    root /var/www/%s/public;
    index index.html index.php;

    access_log /var/log/nginx/%s.access.log;
    error_log  /var/log/nginx/%s.error.log;

    location / {
        try_files $uri $uri/ =404;
    }
}
`, name, name, name, name)
}
