package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets/*
var assets embed.FS

func Handler() http.Handler {
	content, err := fs.Sub(assets, "assets")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(content))
}
