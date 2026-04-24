package handler

import (
	"io/fs"
	"net/http"
	pathpkg "path"
	"strings"

	"study-blocks/web"
)

func RegisterFrontend(mux *http.ServeMux) error {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		return err
	}

	fileServer := http.FileServer(http.FS(dist))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath := strings.TrimPrefix(r.URL.Path, "/")
		if requestedPath == "" {
			http.ServeFileFS(w, r, dist, "index.html")
			return
		}

		if _, err := fs.Stat(dist, requestedPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		if pathpkg.Ext(requestedPath) != "" {
			http.NotFound(w, r)
			return
		}

		http.ServeFileFS(w, r, dist, "index.html")
	}))
	return nil
}
