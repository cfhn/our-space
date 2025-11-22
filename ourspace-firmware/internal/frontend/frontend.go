package frontend

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"
)

//go:generate make -C ../../../ourspace-frontend
//go:generate cp -r ../../../ourspace-frontend/dist/ ./

//go:embed dist/*
var frontend embed.FS

type spaFs struct {
	fs fs.FS
}

func (s spaFs) Open(name string) (fs.File, error) {
	file, err := s.fs.Open(name)
	if errors.Is(err, fs.ErrNotExist) {
		return s.fs.Open("index.html")
	}

	return file, err
}

func ServeFrontend() http.Handler {
	dist, _ := fs.Sub(frontend, "dist")
	return http.FileServer(http.FS(&spaFs{fs: dist}))
}
