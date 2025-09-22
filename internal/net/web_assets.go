package net

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed web/*
var webFS embed.FS

func (s *Server) webFileServer() http.Handler {
	root, err := fs.Sub(webFS, "web")
	if err != nil {
		// this should never happen; panic to surface configuration issue
		panic(err)
	}
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure we always serve index.html for the root path
		if r.URL.Path == "" || r.URL.Path == "/" {
			r = r.Clone(r.Context())
			r.URL.Path = "/index.html"
		}

		// Prevent directory traversal
		if strings.Contains(r.URL.Path, "..") {
			http.NotFound(w, r)
			return
		}

		// Ensure consistent content type for JS modules
		if ext := path.Ext(r.URL.Path); ext == ".js" {
			w.Header().Set("Content-Type", "application/javascript")
		}
		w.Header().Set("Cache-Control", "public, max-age=60")
		fileServer.ServeHTTP(w, r)
	})
}
