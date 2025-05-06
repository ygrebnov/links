package templates

import (
	"embed"
	"io/fs"
)

//go:embed links.html
var linksTemplate embed.FS

func GetLinksTemplate() fs.FS {
	return linksTemplate
}
