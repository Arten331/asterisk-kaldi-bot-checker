package embed

import "embed"

//go:embed *
var memFS embed.FS

func GetEmbedFilesystem() *embed.FS {
	return &memFS
}
