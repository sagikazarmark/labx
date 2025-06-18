package sproutx

import (
	"strings"
	"testing"
	"text/template"

	"github.com/go-sprout/sprout"
	sproutstrings "github.com/go-sprout/sprout/registry/strings"
)

func TestGitHubRegistryIntegration(t *testing.T) {
	// Create a template function map similar to what's done in content.go
	funcs := sprout.New(
		sprout.WithRegistries(
			sproutstrings.NewRegistry(),
			NewFSRegistry(nil), // nil filesystem for this test
			NewStringsRegistry(),
			NewGitHubRegistry(),
		),
	).Build()

	// Test that the GitHub functions are available in the template
	templateStr := `
URL: {{ githubDownloadURL "owner" "repo" "v1.0.0" "asset.tar.gz" }}
`

	tmpl, err := template.New("test").Funcs(funcs).Parse(templateStr)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Execute the template
	var result strings.Builder
	err = tmpl.Execute(&result, nil)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	expected := "URL: https://github.com/owner/repo/releases/download/v1.0.0/asset.tar.gz"
	if !strings.Contains(result.String(), expected) {
		t.Errorf("Expected output to contain %q, got %q", expected, result.String())
	}
}

func TestGitHubRegistryFunctionsAvailable(t *testing.T) {
	// Create a template function map
	funcs := sprout.New(
		sprout.WithRegistries(
			NewGitHubRegistry(),
		),
	).Build()

	// Check that our functions are available
	expectedFuncs := []string{"githubDownloadURL", "githubLatestTag"}
	for _, funcName := range expectedFuncs {
		if _, exists := funcs[funcName]; !exists {
			t.Errorf("Expected function %s to be available in template functions", funcName)
		}
	}
}
