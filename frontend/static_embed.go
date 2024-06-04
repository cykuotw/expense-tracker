package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
)

//go:embed public
var staticEmbedFS embed.FS

type staticFS struct {
	fs fs.FS
}

func (sfs *staticFS) Open(name string) (fs.File, error) {
	return sfs.fs.Open(filepath.Join("public", name))
}

func Static() http.FileSystem {
	staticEmbed := &staticFS{staticEmbedFS}
	return http.FS(staticEmbed)
}
