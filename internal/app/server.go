package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>%s</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; background: #1e1e2e; color: #cdd6f4; }
    h1 { color: #89b4fa; }
    .mermaid { background: #313244; padding: 1rem; border-radius: 8px; }
    .legend { display: flex; gap: 1.5rem; margin-top: 1rem; font-size: 0.9rem; }
    .legend-item { display: flex; align-items: center; gap: 0.4rem; }
    .legend-swatch { width: 14px; height: 14px; border-radius: 3px; }
  </style>
</head>
<body>
  <h1>%s</h1>
  <pre class="mermaid">
%s
  </pre>
  <div class="legend">
    <div class="legend-item"><div class="legend-swatch" style="background:#89b4fa"></div> To Do</div>
    <div class="legend-item"><div class="legend-swatch" style="background:#fab387"></div> In Progress</div>
    <div class="legend-item"><div class="legend-swatch" style="background:#a6e3a1"></div> Done</div>
  </div>
  <script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
    mermaid.initialize({ startOnLoad: true, theme: 'dark' });
  </script>
</body>
</html>`

// DAGServer manages a local HTTP server for rendering the DAG.
type DAGServer struct {
	server *http.Server
	URL    string
}

// StartDAGServer starts an HTTP server on a random port serving the mermaid diagram.
func StartDAGServer(title, mermaidContent string) (*DAGServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("starting listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	html := fmt.Sprintf(htmlTemplate, title, title, mermaidContent)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	server := &http.Server{Handler: mux}

	go server.Serve(listener)

	return &DAGServer{server: server, URL: url}, nil
}

// Shutdown stops the server.
func (s *DAGServer) Shutdown() {
	if s.server != nil {
		s.server.Shutdown(context.Background())
	}
}
