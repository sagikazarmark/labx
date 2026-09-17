package labx

import (
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/sagikazarmark/go-finder"

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

func copyStaticFilesIfExists(root *os.Root, output *os.Root, sourcePath, destPath string) error {
	hasStatic, err := dirExists(root.FS(), sourcePath)
	if err != nil {
		return err
	}

	if !hasStatic {
		return nil
	}

	return copyStaticFiles(root, output, sourcePath, destPath)
}

// dirExists checks if a directory exists
func dirExists(fsys fs.FS, path string) (bool, error) {
	return finder.Exists(fsys, path, finder.FileTypeDir)
}

func fileExists(fsys fs.FS, path string) (bool, error) {
	return finder.Exists(fsys, path, finder.FileTypeFile)
}

func ensureOutputDir(output *os.Root, dirPath string) error {
	if dirPath == "" || dirPath == "." {
		return nil
	}

	current := ""
	for _, part := range strings.Split(dirPath, "/") {
		if part == "" || part == "." {
			continue
		}

		if current == "" {
			current = part
		} else {
			current += "/" + part
		}

		err := output.Mkdir(current, 0o755)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}

	return nil
}

func createOutputFile(output *os.Root, filePath string) (*os.File, error) {
	err := ensureOutputDir(output, path.Dir(filePath))
	if err != nil {
		return nil, err
	}

	return output.Create(filePath)
}

func loadYAMLFile[T any](fsys fs.FS, filePath string) (T, error) {
	file, err := fsys.Open(filePath)
	if err != nil {
		return *new(T), err
	}
	defer file.Close()

	var value T

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&value)

	return value, err
}

func writeStringFile(output *os.Root, filePath, content string) error {
	outputFile, err := createOutputFile(output, filePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	_, err = io.WriteString(outputFile, content)

	return err
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
	outputFile, err := createOutputFile(output, filePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	return writeManifest(outputFile, manifest)
}
