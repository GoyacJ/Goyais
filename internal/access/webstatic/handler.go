package webstatic

import (
	"bytes"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed all:dist
var embeddedFiles embed.FS

type Handler struct {
	files fs.FS
}

func NewHandler() (http.Handler, error) {
	sub, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		return nil, err
	}
	return &Handler{files: sub}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}

	requestPath := path.Clean("/" + strings.TrimSpace(r.URL.Path))
	if strings.HasPrefix(requestPath, "/api/v1/") {
		http.NotFound(w, r)
		return
	}

	if requestPath == "/favicon.ico" || requestPath == "/robots.txt" {
		relative := strings.TrimPrefix(requestPath, "/")
		if h.fileExists(relative) {
			h.serveFile(w, r, relative, false)
			return
		}
		http.NotFound(w, r)
		return
	}

	relative := strings.TrimPrefix(requestPath, "/")
	if relative != "" && h.fileExists(relative) {
		h.serveFile(w, r, relative, relative == "index.html")
		return
	}

	h.serveFile(w, r, "index.html", true)
}

func (h *Handler) fileExists(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	info, err := fs.Stat(h.files, name)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, name string, noStore bool) {
	content, err := fs.ReadFile(h.files, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if noStore {
		w.Header().Set("Cache-Control", "no-store")
	}

	contentType := detectContentType(name, content)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(content))
}

func detectContentType(name string, content []byte) string {
	switch path.Ext(name) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	}
	if byExt := mime.TypeByExtension(path.Ext(name)); byExt != "" {
		return byExt
	}
	if len(content) > 0 {
		return http.DetectContentType(content)
	}
	return "application/octet-stream"
}
