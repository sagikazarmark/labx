package sproutx

import (
	"io/fs"

	"github.com/go-sprout/sprout"
	"github.com/sagikazarmark/labx/pkg/fsx"
)

// FSRegistry struct implements the [sprout.Registry] interface, embedding the Handler to access shared functionalities.
type FSRegistry struct {
	fsys fs.FS

	handler sprout.Handler
}

// NewFSRegistry initializes and returns a new [sprout.Registry].
func NewFSRegistry(fsys fs.FS) *FSRegistry {
	return &FSRegistry{
		fsys: fsys,
	}
}

// Implements [sprout.Registry].
func (r *FSRegistry) UID() string {
	return "sagikazarmark/labx.fs"
}

// Implements [sprout.Registry].
func (r *FSRegistry) LinkHandler(fh sprout.Handler) error {
	r.handler = fh

	return nil
}

// Implements [sprout.Registry].
func (r *FSRegistry) RegisterFunctions(funcsMap sprout.FunctionMap) error {
	sprout.AddFunction(funcsMap, "readFileRange", r.ReadFileRange)

	return nil
}

func (r *FSRegistry) ReadFileRange(name string, from int, to int) (string, error) {
	return fsx.ReadFileRange(r.fsys, name, from, to)
}
