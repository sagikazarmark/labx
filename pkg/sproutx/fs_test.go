package sproutx

import (
	"embed"
	"strings"
	"testing"
	"text/template"

	"github.com/go-sprout/sprout"
)

//go:embed testdata
var testFS embed.FS

func TestFSRegistry_ReadFileBlock(t *testing.T) {
	registry := NewFSRegistry(testFS)

	// Helper function to read expected result from file
	readExpected := func(filename string) string {
		content, err := testFS.ReadFile(filename)
		if err != nil {
			t.Fatalf("Failed to read expected file %s: %v", filename, err)
		}
		return strings.TrimSuffix(string(content), "\n")
	}

	tests := []struct {
		name         string
		fileName     string
		blockName    string
		expectedFile string
		wantErr      bool
	}{
		{
			name:         "read imports block",
			fileName:     "testdata/fs/example.go",
			blockName:    "imports",
			expectedFile: "testdata/fs/expected/imports.txt",
			wantErr:      false,
		},
		{
			name:         "read setup block",
			fileName:     "testdata/fs/example.go",
			blockName:    "setup",
			expectedFile: "testdata/fs/expected/setup.txt",
			wantErr:      false,
		},
		{
			name:         "read cleanup block",
			fileName:     "testdata/fs/example.go",
			blockName:    "cleanup",
			expectedFile: "testdata/fs/expected/cleanup.txt",
			wantErr:      false,
		},
		{
			name:         "read documentation block in multi-line comment",
			fileName:     "testdata/fs/example.go",
			blockName:    "documentation",
			expectedFile: "testdata/fs/expected/documentation.txt",
			wantErr:      false,
		},
		{
			name:         "read constants block with generic endblock",
			fileName:     "testdata/fs/example.go",
			blockName:    "constants",
			expectedFile: "testdata/fs/expected/constants.txt",
			wantErr:      false,
		},
		{
			name:         "bash script variables block",
			fileName:     "testdata/fs/script.sh",
			blockName:    "variables",
			expectedFile: "testdata/fs/expected/bash_variables.txt",
			wantErr:      false,
		},
		{
			name:         "bash script functions block",
			fileName:     "testdata/fs/script.sh",
			blockName:    "functions",
			expectedFile: "testdata/fs/expected/bash_functions.txt",
			wantErr:      false,
		},
		{
			name:      "block not found",
			fileName:  "testdata/fs/example.go",
			blockName: "nonexistent",
			wantErr:   true,
		},
		{
			name:      "file not found",
			fileName:  "testdata/fs/nonexistent.go",
			blockName: "any",
			wantErr:   true,
		},
		{
			name:         "bash script setup block with named endblock",
			fileName:     "testdata/fs/script.sh",
			blockName:    "setup",
			expectedFile: "testdata/fs/expected/bash_setup.txt",
			wantErr:      false,
		},
		{
			name:         "bash script cleanup block with generic endblock",
			fileName:     "testdata/fs/script.sh",
			blockName:    "cleanup",
			expectedFile: "testdata/fs/expected/bash_cleanup.txt",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := registry.ReadFileBlock(tt.fileName, tt.blockName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFileBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				want := readExpected(tt.expectedFile)
				if got != want {
					t.Errorf("ReadFileBlock() = %q, want %q", got, want)
				}
			}
		})
	}
}

func TestExtractBlock(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		blockName string
		want      string
		wantErr   bool
	}{
		{
			name: "simple block",
			content: `line1
// @block:test
content line 1
content line 2
// @endblock:test
line2`,
			blockName: "test",
			want:      "content line 1\ncontent line 2",
			wantErr:   false,
		},
		{
			name: "block with generic endblock",
			content: `line1
# @block:config
key=value
another=line
# @endblock
line2`,
			blockName: "config",
			want:      "key=value\nanother=line",
			wantErr:   false,
		},
		{
			name: "block with whitespace in name",
			content: `line1
<!-- @block:my block -->
<div>content</div>
<!-- @endblock:my block -->
line2`,
			blockName: "my block",
			want:      "<div>content</div>",
			wantErr:   false,
		},
		{
			name: "empty block",
			content: `line1
// @block:empty
// @endblock:empty
line2`,
			blockName: "empty",
			want:      "",
			wantErr:   false,
		},
		{
			name: "unclosed block",
			content: `line1
// @block:unclosed
content line 1
content line 2
line2`,
			blockName: "unclosed",
			want:      "",
			wantErr:   true,
		},
		{
			name: "block not found",
			content: `line1
// @block:other
content
// @endblock:other
line2`,
			blockName: "missing",
			want:      "",
			wantErr:   true,
		},
		{
			name: "multiple blocks with same name (first one returned)",
			content: `line1
// @block:duplicate
first content
// @endblock:duplicate
line2
// @block:duplicate
second content
// @endblock:duplicate
line3`,
			blockName: "duplicate",
			want:      "first content",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractBlock(tt.content, tt.blockName)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFSRegistry_TemplateIntegration(t *testing.T) {
	registry := NewFSRegistry(testFS)

	// Create a sprout function map using the correct API
	funcMap := sprout.New(
		sprout.WithRegistries(registry),
	).Build()

	// Test template with readFileBlock
	tmplText := `{{- readFileBlock "testdata/fs/example.go" "imports" }}`

	tmpl, err := template.New("test").Funcs(funcMap).Parse(tmplText)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, nil)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	// Read expected result from file
	expectedContent, err := testFS.ReadFile("testdata/fs/expected/imports.txt")
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}
	expected := strings.TrimSuffix(string(expectedContent), "\n")

	if buf.String() != expected {
		t.Errorf("Template output = %q, want %q", buf.String(), expected)
	}
}
