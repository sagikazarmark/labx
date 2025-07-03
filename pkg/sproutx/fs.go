package sproutx

import (
	"bufio"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

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
	sprout.AddFunction(funcsMap, "readFile", r.ReadFile)
	sprout.AddFunction(funcsMap, "readFileRange", r.ReadFileRange)
	sprout.AddFunction(funcsMap, "readFileUntil", r.ReadFileUntil)
	sprout.AddFunction(funcsMap, "readFileLine", r.ReadFileLine)
	sprout.AddFunction(funcsMap, "readFileBlock", r.ReadFileBlock)

	return nil
}

func (r *FSRegistry) ReadFile(name string) (string, error) {
	content, err := fs.ReadFile(r.fsys, name)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (r *FSRegistry) ReadFileRange(name string, from int, to int) (string, error) {
	return fsx.ReadFileRange(r.fsys, name, from, to)
}

func (r *FSRegistry) ReadFileUntil(name string, n int) (string, error) {
	return fsx.ReadFileRange(r.fsys, name, 1, n)
}

func (r *FSRegistry) ReadFileLine(name string, n int) (string, error) {
	return fsx.ReadFileRange(r.fsys, name, n, n)
}

// ReadFileBlock reads a named block from a file.
// Block syntax:
//   - Block start: @block:name
//   - Block end: @endblock:name or @endblock
//
// These markers should be embedded in comments appropriate for the file's language.
func (r *FSRegistry) ReadFileBlock(name, blockName string) (string, error) {
	content, err := fs.ReadFile(r.fsys, name)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", name, err)
	}

	return extractBlock(string(content), blockName)
}

// extractBlock parses file content and extracts the named block.
func extractBlock(content, blockName string) (string, error) {
	// Regular expressions to match block start and end markers
	blockStartPattern := regexp.MustCompile(`@block:\s*` + regexp.QuoteMeta(blockName) + `\b`)
	blockEndPattern := regexp.MustCompile(`@endblock(?::\s*` + regexp.QuoteMeta(blockName) + `)?\b`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	var blockLines []string
	inBlock := false
	foundBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		if !inBlock {
			if blockStartPattern.MatchString(line) {
				inBlock = true
				foundBlock = true
				continue // Skip the block start line
			}
		} else {
			if blockEndPattern.MatchString(line) {
				inBlock = false // Mark that we've left the block
				break           // End of block found, stop collecting lines
			}
			blockLines = append(blockLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading content: %w", err)
	}

	if !foundBlock {
		return "", fmt.Errorf("block '%s' not found", blockName)
	}

	if inBlock {
		return "", fmt.Errorf("block '%s' is not properly closed", blockName)
	}

	return strings.Join(blockLines, "\n"), nil
}
