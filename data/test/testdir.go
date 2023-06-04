//go:build test

package testdata

import (
	"embed"
)

//go:embed *
var testFilesystem embed.FS

func GetTestFS() *embed.FS {
	return &testFilesystem
}
