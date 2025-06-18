package sproutx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-sprout/sprout"
)

// GitHubRegistry struct implements the [sprout.Registry] interface, embedding the Handler to access shared functionalities.
type GitHubRegistry struct {
	handler sprout.Handler
}

// NewGitHubRegistry initializes and returns a new [sprout.Registry].
func NewGitHubRegistry() *GitHubRegistry {
	return &GitHubRegistry{}
}

// Implements [sprout.Registry].
func (r *GitHubRegistry) UID() string {
	return "sagikazarmark/labx.github"
}

// Implements [sprout.Registry].
func (r *GitHubRegistry) LinkHandler(fh sprout.Handler) error {
	r.handler = fh

	return nil
}

// Implements [sprout.Registry].
func (r *GitHubRegistry) RegisterFunctions(funcsMap sprout.FunctionMap) error {
	sprout.AddFunction(funcsMap, "githubDownloadURL", r.GitHubDownloadURL)
	sprout.AddFunction(funcsMap, "githubLatestTag", r.GitHubLatestTag)

	return nil
}

// GitHubDownloadURL assembles a GitHub download URL from owner/repository, tag, and asset name.
// Returns a URL in the format: https://github.com/owner/repo/releases/download/tag/asset
func (r *GitHubRegistry) GitHubDownloadURL(owner, repo, tag, asset string) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", owner, repo, tag, asset)
}

// GitHubLatestTag gets the latest tag/release from GitHub for a repository.
// Returns the tag name of the latest release or an error if the request fails.
func (r *GitHubRegistry) GitHubLatestTag(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode GitHub API response: %w", err)
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}
