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

	"maps"

	"github.com/goccy/go-yaml"
)

// loadExtraTemplateData loads additional template data from JSON, YAML, and Markdown files in the data/ directory
func loadExtraTemplateData(fsys fs.FS) (map[string]any, error) {
	return loadExtraTemplateDataFromDirs(fsys, []string{})
}

// loadExtraTemplateDataFromDirs loads additional template data from JSON, YAML, and Markdown files
// from multiple data directories. The root fsys data/ directory has precedence over additional data directories.
func loadExtraTemplateDataFromDirs(rootFS fs.FS, additionalDataDirs []string) (map[string]any, error) {
	// Convert string paths to fs.FS interfaces
	var externalFSs []fs.FS
	for _, dataDir := range additionalDataDirs {
		externalFSs = append(externalFSs, os.DirFS(dataDir))
	}

	return loadExtraTemplateDataFromFilesystems(rootFS, externalFSs)
}

// loadDataFromDir loads data files from a specific directory in a filesystem
func loadDataFromDir(fsys fs.FS, dataDir string) (map[string]any, error) {
	data := make(map[string]any)

	// Check if data directory exists
	_, err := fs.Stat(fsys, dataDir)
	if errors.Is(err, fs.ErrNotExist) {
		// No data directory, return empty map
		return data, nil
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
		var fileDataParsed any
		switch ext {
		case ".json":
			err = json.Unmarshal(fileData, &fileDataParsed)
			if err != nil {
				return fmt.Errorf("parse JSON file %s: %w", path, err)
			}
		case ".yaml", ".yml":
			err = yaml.Unmarshal(fileData, &fileDataParsed)
			if err != nil {
				return fmt.Errorf("parse YAML file %s: %w", path, err)
			}
		case ".md", ".markdown":
			// Store markdown files as strings
			fileDataParsed = string(fileData)
		}

		// Use filename without extension as key
		key := strings.TrimSuffix(d.Name(), originalExt)
		data[key] = fileDataParsed

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk data directory: %w", err)
	}

	return data, nil
}

// loadExtraTemplateDataFromFilesystems loads data from multiple fs.FS instances for testing
func loadExtraTemplateDataFromFilesystems(rootFS fs.FS, externalFSs []fs.FS) (map[string]any, error) {
	extraData := make(map[string]any)

	// Load from external filesystems first (lower precedence)
	for _, externalFS := range externalFSs {
		dirData, err := loadDataFromDir(externalFS, ".")
		if err != nil {
			// Skip filesystems that can't be read
			continue
		}

		// Merge data (files loaded later will overwrite)
		maps.Copy(extraData, dirData)
	}

	// Load from root filesystem data/ directory last (highest precedence)
	rootData, err := loadDataFromDir(rootFS, "data")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("load data from root data directory: %w", err)
	}

	// Merge root data (will overwrite any conflicting keys)
	maps.Copy(extraData, rootData)

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
