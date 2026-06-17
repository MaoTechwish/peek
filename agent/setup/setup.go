package setup

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"text/template"
)

const setupHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Peek Setup</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; max-width: 560px; margin: 80px auto; padding: 0 24px; color: #222; }
  h1 { font-size: 20px; font-weight: 600; margin-bottom: 8px; }
  p { color: #555; margin-bottom: 16px; font-size: 14px; }
  .url { background: #f5f5f5; border: 1px solid #ddd; border-radius: 6px; padding: 12px 16px; font-family: monospace; font-size: 13px; word-break: break-all; margin-bottom: 16px; }
  .actions { display: flex; gap: 10px; }
  button { padding: 9px 20px; font-size: 14px; border-radius: 6px; border: none; cursor: pointer; }
  .copy { background: #f0f0f0; color: #333; }
  .dismiss { background: #222; color: #fff; }
  .copy:hover { background: #e0e0e0; }
  .dismiss:hover { background: #000; }
</style>
</head>
<body>
<h1>Peek is ready</h1>
<p>Share this URL to let someone view your screen:</p>
<div class="url" id="url">{{.ViewerURL}}</div>
<div class="actions">
  <button class="copy" onclick="navigator.clipboard.writeText(document.getElementById('url').textContent).then(()=>{this.textContent='Copied!'})">Copy link</button>
  <button class="dismiss" onclick="fetch('/dismiss').then(()=>window.close())">Start &amp; Dismiss</button>
</div>
</body>
</html>`

// Show opens a local browser tab displaying viewerURL, then blocks until the user clicks "Start & Dismiss".
// If the browser cannot be opened, the URL is printed to stdout as a fallback.
func Show(viewerURL string) {
	tmpl := template.Must(template.New("setup").Parse(setupHTML))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("\nPeek viewer URL: %s\n\nPress Enter after you have copied the URL...\n", viewerURL)
		fmt.Scanln()
		return
	}

	done := make(chan struct{})
	srv := &http.Server{}
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, struct{ ViewerURL string }{viewerURL})
	})
	mux.HandleFunc("/dismiss", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		select {
		case <-done:
		default:
			close(done)
		}
	})
	srv.Handler = mux

	go srv.Serve(ln)

	addr := fmt.Sprintf("http://127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port)
	if err := openBrowser(addr); err != nil {
		fmt.Printf("\nOpen this URL to copy your viewer link: %s\n", addr)
	}

	<-done
	srv.Shutdown(context.Background())
}
