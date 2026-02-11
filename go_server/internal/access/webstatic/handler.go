// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package webstatic

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var embeddedDist embed.FS

type Handler struct {
	root fs.FS
}

var (
	fallbackIndexHTML = []byte(`<!doctype html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Goyais</title></head><body><div id="app">Goyais</div><script type="module" src="/assets/app.js"></script></body></html>`)
	fallbackAppJS     = []byte(`console.log("goyais fallback static bundle");`)
)

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
		if h.serveFallbackStatic(w, r, requested) {
			return
		}

		if strings.HasPrefix(requested, "assets/") || path.Ext(requested) != "" {
			http.NotFound(w, r)
			return
		}
	}

	if !h.serveIfExists(w, r, "index.html", true) {
		h.serveFallbackIndex(w, r)
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

func (h *Handler) serveFallbackStatic(w http.ResponseWriter, r *http.Request, name string) bool {
	if strings.HasPrefix(name, "assets/") && strings.HasSuffix(name, ".js") {
		w.Header().Set("Content-Type", "application/javascript")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write(fallbackAppJS)
		}
		return true
	}
	return false
}

func (h *Handler) serveFallbackIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(fallbackIndexHTML)
	}
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
