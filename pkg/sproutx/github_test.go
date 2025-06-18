package sproutx

import (
	"testing"
)

func TestGitHubRegistry_GitHubDownloadURL(t *testing.T) {
	registry := NewGitHubRegistry()

	tests := []struct {
		name     string
		owner    string
		repo     string
		tag      string
		asset    string
		expected string
	}{
		{
			name:     "basic URL construction",
			owner:    "owner",
			repo:     "repo",
			tag:      "v1.0.0",
			asset:    "binary.tar.gz",
			expected: "https://github.com/owner/repo/releases/download/v1.0.0/binary.tar.gz",
		},
		{
			name:     "complex repository name",
			owner:    "kubernetes",
			repo:     "kubectl",
			tag:      "v1.28.0",
			asset:    "kubectl-linux-amd64.tar.gz",
			expected: "https://github.com/kubernetes/kubectl/releases/download/v1.28.0/kubectl-linux-amd64.tar.gz",
		},
		{
			name:     "organization with dashes",
			owner:    "go-sprout",
			repo:     "sprout",
			tag:      "v0.5.1",
			asset:    "sprout_0.5.1_linux_amd64.tar.gz",
			expected: "https://github.com/go-sprout/sprout/releases/download/v0.5.1/sprout_0.5.1_linux_amd64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.GitHubDownloadURL(tt.owner, tt.repo, tt.tag, tt.asset)
			if result != tt.expected {
				t.Errorf("GitHubDownloadURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGitHubRegistry_UID(t *testing.T) {
	registry := NewGitHubRegistry()
	expected := "sagikazarmark/labx.github"

	if registry.UID() != expected {
		t.Errorf("UID() = %v, want %v", registry.UID(), expected)
	}
}

func TestGitHubRegistry_LinkHandler(t *testing.T) {
	registry := NewGitHubRegistry()

	// Test that LinkHandler doesn't return an error
	err := registry.LinkHandler(nil)
	if err != nil {
		t.Errorf("LinkHandler() returned unexpected error: %v", err)
	}
}

func TestGitHubRegistry_RegisterFunctions(t *testing.T) {
	registry := NewGitHubRegistry()
	funcsMap := make(map[string]interface{})

	err := registry.RegisterFunctions(funcsMap)
	if err != nil {
		t.Errorf("RegisterFunctions() returned unexpected error: %v", err)
	}

	// Check that the expected functions were registered
	expectedFuncs := []string{"githubDownloadURL", "githubLatestTag"}
	for _, funcName := range expectedFuncs {
		if _, exists := funcsMap[funcName]; !exists {
			t.Errorf("Expected function %s was not registered", funcName)
		}
	}
}

// Note: GitHubLatestTag function is not tested here as it makes actual HTTP requests
// In a real-world scenario, you would want to mock the HTTP client or use dependency injection
// to make this function testable without making actual network calls.
