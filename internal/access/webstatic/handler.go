package webstatic

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

//go:embed dist
var embeddedDist embed.FS

type Handler struct {
	root fs.FS
}

func NewHandler() (http.Handler, error) {
	sub, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return nil, err
	}
	return &Handler{root: sub}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requested := cleanPath(r.URL.Path)

	if requested == "favicon.ico" || requested == "robots.txt" {
		if h.serveIfExists(w, r, requested, false) {
			return
		}
		http.NotFound(w, r)
		return
	}

	if requested != "" {
		if h.serveIfExists(w, r, requested, false) {
			return
		}

		if strings.HasPrefix(requested, "assets/") || path.Ext(requested) != "" {
			http.NotFound(w, r)
			return
		}
	}

	if !h.serveIfExists(w, r, "index.html", true) {
		http.NotFound(w, r)
	}
}

func cleanPath(raw string) string {
	clean := path.Clean("/" + strings.TrimSpace(raw))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "." {
		return ""
	}
	return clean
}

func (h *Handler) serveIfExists(w http.ResponseWriter, r *http.Request, name string, forceNoStore bool) bool {
	info, err := fs.Stat(h.root, name)
	if err != nil || info.IsDir() {
		return false
	}

	payload, err := fs.ReadFile(h.root, name)
	if err != nil {
		return false
	}

	contentType := detectContentType(name, payload)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	if forceNoStore || strings.EqualFold(name, "index.html") {
		w.Header().Set("Cache-Control", "no-store")
	}

	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(payload)
	}
	return true
}

func detectContentType(name string, payload []byte) string {
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	}

	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}

	if len(payload) == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(payload)
}
