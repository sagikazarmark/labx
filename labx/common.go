package labx

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/sagikazarmark/labx/core"
)

const defaultImageRepo = "ghcr.io/sagikazarmark/iximiuz-labs"

const betaNotice = `::remark-box
---
kind: warning
---

⚠️ This content is marked as **beta**, meaning it's unfinished or still in progress and may change significantly.
::

`

func hasFiles(fsys fs.FS, kind content.ContentKind) (bool, error) {
	_, err := fs.Stat(fsys, fmt.Sprintf("dist/__static__/%s.tar.gz", kind.String()))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func createDownloadScript(kind content.ContentKind) string {
	targetDir := fmt.Sprintf("/opt/%s", kind)
	url := fmt.Sprintf("https://labs.iximiuz.com/__static__/%s.tar.gz?t=$(date +%%s)", kind)

	return fmt.Sprintf(
		"mkdir -p %s\nwget --no-cache -O - \"%s\" | tar -xz -C %s",
		targetDir,
		url,
		targetDir,
	)
}

// copyStaticFiles copies static files from source to destination
func copyStaticFiles(root *os.Root, output *os.Root, sourcePath, destPath string) error {
	fsys := root.FS()

	// Create the parent static directory first
	err := output.Mkdir(destPath, 0o755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	return fs.WalkDir(fsys, sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourcePath {
			return nil
		}

		// Calculate relative path from source
		relPath := strings.TrimPrefix(path, sourcePath+"/")
		outputPath := destPath + "/" + relPath

		if d.IsDir() {
			// Create directory in destination
			err = output.Mkdir(outputPath, 0o755)
			if err != nil && !os.IsExist(err) {
				return err
			}
			return nil
		}

		// Copy file
		sourceFile, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		destFile, err := output.Create(outputPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, sourceFile)
		return err
	})
}

// dirExists checks if a directory exists
func dirExists(fsys fs.FS, path string) (bool, error) {
	stat, err := fs.Stat(fsys, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return stat.IsDir(), nil
}

func fileExists(fsys fs.FS, path string) (bool, error) {
	stat, err := fs.Stat(fsys, path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if stat.IsDir() {
		return false, nil
	}

	return true, nil
}

func writeManifest[T api.PlaygroundManifest | core.ContentManifest](w io.Writer, manifest T) error {
	_, isContent := any(manifest).(core.ContentManifest)
	if isContent {
		w = newFrontMatterWriter(w)
	}

	encoder := yaml.NewEncoder(
		w,
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.IndentSequence(true),
	)

	return encoder.Encode(manifest)
}

func renderManifest[T api.PlaygroundManifest | core.ContentManifest](
	output *os.Root,
	filePath string,
	manifest T,
) error {
	outputFile, err := output.Create(filePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	return writeManifest(outputFile, manifest)
}
