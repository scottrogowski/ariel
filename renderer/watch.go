package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/scottmrogowski/ariel/dsl"
)

// wsMessage is the JSON structure sent to browser clients.
type wsMessage struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"` // populated on "update"
	Message string `json:"message,omitempty"` // populated on "error"
}

// WatchServer holds state for a running watch server.
type WatchServer struct {
	port     int
	filePath string

	mu      sync.RWMutex
	html    string // current rendered HTML (includes WS snippet)
	clients map[*websocket.Conn]struct{}

	upgrader websocket.Upgrader
}

// NewWatchServer creates a WatchServer for the given file and port.
func NewWatchServer(filePath string, port int, initialHTML string) *WatchServer {
	return &WatchServer{
		port:     port,
		filePath: filePath,
		html:     initialHTML,
		clients:  make(map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// UpdateContent re-renders and broadcasts an update to all connected clients.
func (s *WatchServer) UpdateContent(w *dsl.Walkthrough) {
	html, err := render(w, s.wsSnippet())
	if err != nil {
		s.broadcastError(fmt.Sprintf("render error: %v", err))
		return
	}

	s.mu.Lock()
	s.html = html
	s.mu.Unlock()

	msg, _ := json.Marshal(wsMessage{Type: "update", Content: html})
	s.broadcast(msg)
}

// BroadcastError sends an error message to all connected clients.
func (s *WatchServer) BroadcastError(text string) {
	s.broadcastError(text)
}

func (s *WatchServer) broadcastError(text string) {
	msg, _ := json.Marshal(wsMessage{Type: "error", Message: text})
	s.broadcast(msg)
}

func (s *WatchServer) broadcast(msg []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for conn := range s.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("ws write error: %v", err)
		}
	}
}

// Start starts the HTTP server and blocks until ctx is cancelled.
func (s *WatchServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/", s.handlePage)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("port %d already in use", s.port)
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	return srv.Serve(ln)
}

func (s *WatchServer) handlePage(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	html := s.html
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

func (s *WatchServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	s.mu.Lock()
	s.clients[conn] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	// Read loop — we only need to handle pings/close frames.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// wsSnippet returns the websocket client JS to inject before </body>.
func (s *WatchServer) wsSnippet() string {
	return strings.ReplaceAll(`<script>
(function() {
  var ws = new WebSocket('ws://localhost:PORT/ws');
  var overlay = null;

  function showError(msg) {
    if (!overlay) {
      overlay = document.createElement('div');
      overlay.style.cssText = 'position:fixed;bottom:20px;left:50%;transform:translateX(-50%);background:#7f1d1d;color:#fca5a5;padding:12px 20px;border-radius:8px;font-family:monospace;font-size:13px;max-width:80%;z-index:9999;border:1px solid #991b1b;';
      document.body.appendChild(overlay);
    }
    overlay.textContent = '⚠ ' + msg;
  }

  function clearError() {
    if (overlay) { overlay.remove(); overlay = null; }
  }

  ws.onmessage = function(e) {
    var msg = JSON.parse(e.data);
    if (msg.type === 'update') {
      clearError();
      document.open();
      document.write(msg.content);
      document.close();
    } else if (msg.type === 'error') {
      showError(msg.message);
    }
  };

  ws.onclose = function() {
    showError('connection lost — save the file to reconnect');
  };
})();
</script>
`, "PORT", fmt.Sprintf("%d", s.port))
}
