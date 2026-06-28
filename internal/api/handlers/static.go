package handlers

import (
	"io/fs"
	"net/http"
	"strings"
)

// SPAHandler serves the embedded React build and falls back to index.html for
// any path that doesn't match a real asset (client-side routing).
func SPAHandler(fsys fs.FS) http.HandlerFunc {
	fsServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to serve the file from the embedded FS
		if _, err := fs.Stat(fsys, path); err == nil {
			fsServer.ServeHTTP(w, r)
			return
		}

		// Fall back to index.html so React Router handles unknown paths
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/index.html"
		fsServer.ServeHTTP(w, r2)
	}
}
