package fsx

import (
	"fmt"
	"io/fs"

	"github.com/sagikazarmark/labx/pkg/iox"
)

// ReadFileRange reads lines from a file in [fs.FS] in the specified range (or until EOF).
func ReadFileRange(fsys fs.FS, name string, from, to int) (string, error) {
	file, err := fsys.Open(name)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return iox.ReadLinesRange(file, from, to)
}
