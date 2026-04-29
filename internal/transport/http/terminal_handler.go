package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/your-org/ventopanel/internal/domain/server"
	"github.com/your-org/ventopanel/internal/filemanager"
)

// ─── WebSocket upgrader ───────────────────────────────────────────────────────

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Auth is already validated by the JWT middleware before we reach here.
	CheckOrigin: func(_ *http.Request) bool { return true },
}

// ─── Handler ──────────────────────────────────────────────────────────────────

type serverGetter interface {
	GetByID(ctx context.Context, id string) (*server.Server, error)
}

type TerminalHandler struct {
	servers serverGetter
}

func NewTerminalHandler(servers serverGetter) *TerminalHandler {
	return &TerminalHandler{servers: servers}
}

// resizePayload is the JSON message the frontend sends when the terminal is resized.
type resizePayload struct {
	Cols uint32 `json:"cols"`
	Rows uint32 `json:"rows"`
}

// Connect handles GET /servers/:id/terminal (WebSocket upgrade).
//
// Protocol (both directions use the WebSocket binary/text frame type):
//   - Binary frame  → raw terminal bytes (stdin input or stdout/stderr output)
//   - Text frame    → JSON resize message  {"cols":80,"rows":24}
func (h *TerminalHandler) Connect(c *gin.Context) {
	serverID := c.Param("id")

	// Guard: must be authenticated (middleware already set the user ID if valid).
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	srv, err := h.servers.GetByID(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "server not found"})
		return
	}

	// Dial SSH — use a fresh connection per terminal session (not the SFTP pool).
	sshCli, err := filemanager.DialSSH(filemanager.ServerDialConfig{
		Host:     srv.Host,
		Port:     srv.Port,
		User:     srv.SSHUser,
		Password: srv.SSHPassword,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, errorResponse{Error: "SSH: " + err.Error()})
		return
	}
	defer sshCli.Close()

	// Upgrade to WebSocket.
	ws, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade writes the error response itself.
		return
	}
	defer ws.Close()

	// Open SSH session with PTY.
	sess, err := sshCli.NewSession()
	if err != nil {
		writeWSText(ws, "\r\n\x1b[31mFailed to open SSH session: "+err.Error()+"\x1b[0m\r\n")
		return
	}
	defer sess.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 38400,
		ssh.TTY_OP_OSPEED: 38400,
	}
	if err := sess.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		writeWSText(ws, "\r\n\x1b[31mPTY request failed: "+err.Error()+"\x1b[0m\r\n")
		return
	}

	stdin, err := sess.StdinPipe()
	if err != nil {
		writeWSText(ws, "\r\n\x1b[31m"+err.Error()+"\x1b[0m\r\n")
		return
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		writeWSText(ws, "\r\n\x1b[31m"+err.Error()+"\x1b[0m\r\n")
		return
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		writeWSText(ws, "\r\n\x1b[31m"+err.Error()+"\x1b[0m\r\n")
		return
	}

	if err := sess.Shell(); err != nil {
		writeWSText(ws, "\r\n\x1b[31mShell start failed: "+err.Error()+"\x1b[0m\r\n")
		return
	}

	// Mutex to serialise concurrent WebSocket writes.
	var wsMu sync.Mutex
	writeWSBin := func(data []byte) {
		wsMu.Lock()
		defer wsMu.Unlock()
		_ = ws.WriteMessage(websocket.BinaryMessage, data)
	}
	writeDone := make(chan struct{})

	// SSH stdout → WebSocket binary frames.
	go pipeToWS(stdout, writeWSBin, writeDone)
	// SSH stderr → WebSocket binary frames (merged into same stream).
	go pipeToWS(stderr, writeWSBin, writeDone)

	// WebSocket → SSH stdin (read loop; also handles resize messages).
	for {
		mt, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}
		switch mt {
		case websocket.BinaryMessage:
			if _, err := stdin.Write(msg); err != nil {
				goto done
			}
		case websocket.TextMessage:
			var r resizePayload
			if json.Unmarshal(msg, &r) == nil && r.Cols > 0 && r.Rows > 0 {
				_ = sess.WindowChange(int(r.Rows), int(r.Cols))
			} else {
				// Could be a text keystroke (some clients send text frames).
				if _, err := stdin.Write(msg); err != nil {
					goto done
				}
			}
		}
	}

done:
	stdin.Close()
	sess.Wait() //nolint:errcheck
	<-writeDone
	<-writeDone
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func pipeToWS(r io.Reader, write func([]byte), done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			cp := make([]byte, n)
			copy(cp, buf[:n])
			write(cp)
		}
		if err != nil {
			return
		}
	}
}

func writeWSText(ws *websocket.Conn, text string) {
	_ = ws.WriteMessage(websocket.TextMessage, []byte(text))
}
