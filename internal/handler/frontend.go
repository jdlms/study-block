package handler

import (
	"io/fs"
	"net/http"
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
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			http.ServeFileFS(w, r, dist, "index.html")
			return
		}

		if _, err := fs.Stat(dist, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFileFS(w, r, dist, "index.html")
	}))
	return nil
}
