package labx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
)

// loadExtraTemplateData loads additional template data from JSON, YAML, and Markdown files in the data/ directory
func loadExtraTemplateData(fsys fs.FS) (map[string]any, error) {
	extraData := make(map[string]any)

	const dataDir = "data"

	// Check if data directory exists
	_, err := fs.Stat(fsys, dataDir)
	if errors.Is(err, fs.ErrNotExist) {
		// No data directory, return empty map
		return extraData, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat data directory: %w", err)
	}

	// Walk through the data directory
	err = fs.WalkDir(fsys, dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		// TODO: consider supporting nested data
		if d.IsDir() {
			return nil
		}

		// Only process JSON, YAML, and Markdown files
		originalExt := filepath.Ext(d.Name())
		ext := strings.ToLower(originalExt)
		if !slices.Contains([]string{".json", ".yaml", ".yml", ".md", ".markdown"}, ext) {
			return nil
		}

		// Read the file
		fileData, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", path, err)
		}

		// Parse based on extension
		var data any
		switch ext {
		case ".json":
			err = json.Unmarshal(fileData, &data)
			if err != nil {
				return fmt.Errorf("parse JSON file %s: %w", path, err)
			}
		case ".yaml", ".yml":
			err = yaml.Unmarshal(fileData, &data)
			if err != nil {
				return fmt.Errorf("parse YAML file %s: %w", path, err)
			}
		case ".md", ".markdown":
			// Store markdown files as strings
			data = string(fileData)
		}

		// Use filename without extension as key
		key := strings.TrimSuffix(d.Name(), originalExt)
		extraData[key] = data

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk data directory: %w", err)
	}

	return extraData, nil
}

func renderRootTemplate(ctx renderContext, tpl *template.Template, name string) error {
	data := templateData{
		Channel:  ctx.Channel,
		Manifest: ctx.Manifest,
		Extra:    ctx.Extra,
	}

	return renderTemplate(ctx.Output, name, tpl, name, data)
}

func renderTemplate(
	output *os.Root,
	outputPath string,
	tpl *template.Template,
	name string,
	data any,
) error {
	outputFile, err := output.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	return tpl.ExecuteTemplate(outputFile, name, data)
}
