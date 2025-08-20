package labx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

// loadAllExtraData loads data from multiple fs.FS instances for testing
func loadAllExtraData(rootFS fs.FS, dataFSs []fs.FS) (map[string]any, error) {
	extraData := make(map[string]any)

	// Load from external filesystems first (lower precedence)
	for _, dataFS := range dataFSs {
		dirData, err := loadExtraData(dataFS, ".")
		if err != nil {
			// Skip filesystems that can't be read
			continue
		}

		// Merge data (files loaded later will overwrite)
		maps.Copy(extraData, dirData)
	}

	// Load from root filesystem data/ directory last (highest precedence)
	rootData, err := loadExtraData(rootFS, "data")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("load data from root data directory: %w", err)
	}

	// Merge root data (will overwrite any conflicting keys)
	maps.Copy(extraData, rootData)

	return extraData, nil
}

// loadExtraData loads data files from a specific directory in a filesystem
func loadExtraData(fsys fs.FS, dataDir string) (map[string]any, error) {
	data := make(map[string]any)

	// Walk through the data directory
	err := fs.WalkDir(fsys, dataDir, func(path string, d fs.DirEntry, err error) error {
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
	if errors.Is(err, fs.ErrNotExist) {
		// No data directory, return empty map
		return data, nil
	}
	if err != nil {
		return nil, fmt.Errorf("walk data directory: %w", err)
	}

	return data, nil
}
