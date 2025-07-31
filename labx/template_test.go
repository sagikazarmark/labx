package labx

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadExtraTemplateData(t *testing.T) {
	testCases := []struct {
		name     string
		fsys     fstest.MapFS
		expected map[string]any
		wantErr  bool
	}{
		{
			name: "no data directory",
			fsys: fstest.MapFS{
				"manifest.yaml": &fstest.MapFile{Data: []byte("kind: course")},
			},
			expected: map[string]any{},
			wantErr:  false,
		},
		{
			name: "empty data directory",
			fsys: fstest.MapFS{
				"data/.gitkeep": &fstest.MapFile{Data: []byte("")},
			},
			expected: map[string]any{},
			wantErr:  false,
		},
		{
			name: "single JSON file",
			fsys: fstest.MapFS{
				"data/config.json": &fstest.MapFile{Data: []byte(`{"name": "test", "version": 1}`)},
			},
			expected: map[string]any{
				"config": map[string]any{
					"name":    "test",
					"version": float64(1),
				},
			},
			wantErr: false,
		},
		{
			name: "single YAML file",
			fsys: fstest.MapFS{
				"data/settings.yaml": &fstest.MapFile{Data: []byte("name: test\nversion: 1\nenabled: true")},
			},
			expected: map[string]any{
				"settings": map[string]any{
					"name":    "test",
					"version": uint64(1),
					"enabled": true,
				},
			},
			wantErr: false,
		},
		{
			name: "markdown files with different extensions",
			fsys: fstest.MapFS{
				"data/readme.md":     &fstest.MapFile{Data: []byte("# Welcome\n\nThis is a readme file.")},
				"data/docs.markdown": &fstest.MapFile{Data: []byte("## Documentation\n\nSome docs here.")},
				"data/empty.md":      &fstest.MapFile{Data: []byte("")},
			},
			expected: map[string]any{
				"readme": "# Welcome\n\nThis is a readme file.",
				"docs":   "## Documentation\n\nSome docs here.",
				"empty":  "",
			},
			wantErr: false,
		},
		{
			name: "mixed file types including markdown",
			fsys: fstest.MapFS{
				"data/config.json":   &fstest.MapFile{Data: []byte(`{"api_url": "https://api.example.com"}`)},
				"data/features.yaml": &fstest.MapFile{Data: []byte("feature_a: true\nfeature_b: false")},
				"data/metadata.yml":  &fstest.MapFile{Data: []byte("author: John Doe\ncreated: 2024-01-01")},
				"data/readme.md":     &fstest.MapFile{Data: []byte("# Project\nDocumentation here")},
				"data/ignored.txt":   &fstest.MapFile{Data: []byte("this should be ignored")},
			},
			expected: map[string]any{
				"config": map[string]any{
					"api_url": "https://api.example.com",
				},
				"features": map[string]any{
					"feature_a": true,
					"feature_b": false,
				},
				"metadata": map[string]any{
					"author":  "John Doe",
					"created": "2024-01-01",
				},
				"readme": "# Project\nDocumentation here",
			},
			wantErr: false,
		},
		{
			name: "nested directory structure",
			fsys: fstest.MapFS{
				"data/global.json":           &fstest.MapFile{Data: []byte(`{"theme": "dark"}`)},
				"data/nested/local.yaml":     &fstest.MapFile{Data: []byte("debug: true")},
				"data/deeply/nested/app.yml": &fstest.MapFile{Data: []byte("name: myapp")},
				"data/docs/api.md":           &fstest.MapFile{Data: []byte("# API\nDocumentation")},
			},
			expected: map[string]any{
				"global": map[string]any{
					"theme": "dark",
				},
				"local": map[string]any{
					"debug": true,
				},
				"app": map[string]any{
					"name": "myapp",
				},
				"api": "# API\nDocumentation",
			},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			fsys: fstest.MapFS{
				"data/invalid.json": &fstest.MapFile{Data: []byte(`{"invalid": json}`)},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "invalid YAML",
			fsys: fstest.MapFS{
				"data/invalid.yaml": &fstest.MapFile{Data: []byte("invalid: yaml: content: [")},
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := loadExtraTemplateData(testCase.fsys)

			if testCase.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestLoadExtraTemplateDataSpecialCases(t *testing.T) {
	t.Run("filename parsing with various patterns", func(t *testing.T) {
		fsys := fstest.MapFS{
			"data/my-config.json":      &fstest.MapFile{Data: []byte(`{"key": "value1"}`)},
			"data/my_settings.yaml":    &fstest.MapFile{Data: []byte("key: value2")},
			"data/app.config.yml":      &fstest.MapFile{Data: []byte("key: value3")},
			"data/file.with.dots.json": &fstest.MapFile{Data: []byte(`{"key": "value4"}`)},
			"data/guide.md":            &fstest.MapFile{Data: []byte("# Guide\nContent here")},
			"data/notes.markdown":      &fstest.MapFile{Data: []byte("## Notes\nSome notes")},
		}

		result, err := loadExtraTemplateData(fsys)
		require.NoError(t, err)

		expected := map[string]any{
			"my-config": map[string]any{
				"key": "value1",
			},
			"my_settings": map[string]any{
				"key": "value2",
			},
			"app.config": map[string]any{
				"key": "value3",
			},
			"file.with.dots": map[string]any{
				"key": "value4",
			},
			"guide": "# Guide\nContent here",
			"notes": "## Notes\nSome notes",
		}

		assert.Equal(t, expected, result)
	})

	t.Run("case insensitive extensions", func(t *testing.T) {
		fsys := fstest.MapFS{
			"data/markdown1.MD":       &fstest.MapFile{Data: []byte("# Upper case MD")},
			"data/markdown2.Markdown": &fstest.MapFile{Data: []byte("# Mixed case Markdown")},
			"data/config1.JSON":       &fstest.MapFile{Data: []byte(`{"case": "upper"}`)},
			"data/config2.YAML":       &fstest.MapFile{Data: []byte("case: upper")},
		}

		result, err := loadExtraTemplateData(fsys)
		require.NoError(t, err)

		assert.Equal(t, "# Upper case MD", result["markdown1"])
		assert.Equal(t, "# Mixed case Markdown", result["markdown2"])
		assert.Equal(t, map[string]any{"case": "upper"}, result["config1"])
		assert.Equal(t, map[string]any{"case": "upper"}, result["config2"])
	})

	t.Run("markdown never errors", func(t *testing.T) {
		fsys := fstest.MapFS{
			"data/any-content.md": &fstest.MapFile{Data: []byte("This could be anything: {[}]@#$%^&*()")},
			"data/unicode.md":     &fstest.MapFile{Data: []byte("🚀 Unicode: αβγ\n\nCode: `console.log('hello');`")},
			"data/whitespace.md":  &fstest.MapFile{Data: []byte("   \n\t\n   ")},
		}

		result, err := loadExtraTemplateData(fsys)
		require.NoError(t, err)

		assert.Equal(t, "This could be anything: {[}]@#$%^&*()", result["any-content"])
		assert.Equal(t, "🚀 Unicode: αβγ\n\nCode: `console.log('hello');`", result["unicode"])
		assert.Equal(t, "   \n\t\n   ", result["whitespace"])
	})
}

func TestLoadExtraTemplateDataPerformance(t *testing.T) {
	// Create a filesystem with many files to test performance
	fsys := fstest.MapFS{}

	// Add 50 files of each type for performance testing
	for i := range 50 {
		fsys[fmt.Sprintf("data/config%d.json", i)] = &fstest.MapFile{
			Data: fmt.Appendf(nil, `{"id": %d, "name": "config%d"}`, i, i),
		}
		fsys[fmt.Sprintf("data/settings%d.yaml", i)] = &fstest.MapFile{
			Data: fmt.Appendf(nil, "id: %d\nname: settings%d", i, i),
		}
		fsys[fmt.Sprintf("data/doc%d.md", i)] = &fstest.MapFile{
			Data: fmt.Appendf(nil, "# Document %d\n\nContent for document %d", i, i),
		}
	}

	result, err := loadExtraTemplateData(fsys)
	require.NoError(t, err)

	// Should have 150 entries (50 of each type)
	assert.Len(t, result, 150)

	// Verify some samples
	assert.Equal(t, map[string]any{"id": float64(0), "name": "config0"}, result["config0"])
	assert.Equal(t, map[string]any{"id": uint64(25), "name": "settings25"}, result["settings25"])
	assert.Equal(t, "# Document 49\n\nContent for document 49", result["doc49"])
}
