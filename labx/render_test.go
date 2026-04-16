package labx_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/stretchr/testify/require"

	"github.com/sagikazarmark/labx/labx"
)

func TestRenderPlaygroundMatchesGenerateMarkdown(t *testing.T) {
	t.Parallel()

	generateDir := t.TempDir()
	renderDir := t.TempDir()
	templateDir := t.TempDir()
	dataDir := t.TempDir()

	writeTestFile(
		t,
		filepath.Join(templateDir, "shared.md"),
		`{{- define "shared" -}}Shared: {{ . }}{{- end -}}`,
	)
	writeTestFile(t, filepath.Join(dataDir, "note.md"), "global")
	writeTestFile(t, filepath.Join(generateDir, "data", "note.md"), "root")

	writeTestFile(t, filepath.Join(generateDir, "README.md"), `# {{ .Manifest.Title }}

Name: {{ .Manifest.Name }}
Channel: {{ .Channel }}
Note: {{ .Extra.note }}
{{ template "shared" .Manifest.Name }}
`)
	writeTestFile(t, filepath.Join(generateDir, "manifest.yaml"), `kind: playground
name: demo
title: Demo Playground
description: Demo playground
channels:
  dev:
    name: demo-dev
playground:
  tabs: []
  machines: []
`)

	writeTestFile(t, filepath.Join(renderDir, "README.md"), `# {{ .Manifest.Title }}

Name: {{ .Manifest.Name }}
Channel: {{ .Channel }}
Note: {{ .Extra.note }}
{{ template "shared" .Manifest.Name }}
`)
	writeTestFile(t, filepath.Join(renderDir, "manifest.yaml"), `kind: playground
name: demo-dev
title: "DEV: Demo Playground"
description: Demo playground
playground:
  tabs: []
  machines: []
`)

	generateRoot, err := os.OpenRoot(generateDir)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(generateDir, "out"), 0o755)
	require.NoError(t, err)

	generateOutput, err := os.OpenRoot(filepath.Join(generateDir, "out"))
	require.NoError(t, err)

	err = labx.Generate(labx.GenerateOpts{
		Root:         generateRoot,
		Output:       generateOutput,
		Channel:      "dev",
		TemplateDirs: []fs.FS{os.DirFS(templateDir)},
		DataDirs:     []fs.FS{os.DirFS(dataDir)},
	})
	require.NoError(t, err)

	generatedManifestBytes, err := os.ReadFile(filepath.Join(generateDir, "out", "manifest.yaml"))
	require.NoError(t, err)

	var generatedManifest api.PlaygroundManifest
	err = yaml.Unmarshal(generatedManifestBytes, &generatedManifest)
	require.NoError(t, err)

	writeTestFile(t, filepath.Join(renderDir, "data", "note.md"), "root")

	renderRoot, err := os.OpenRoot(renderDir)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(renderDir, "out"), 0o755)
	require.NoError(t, err)

	renderOutput, err := os.OpenRoot(filepath.Join(renderDir, "out"))
	require.NoError(t, err)

	err = labx.Render(labx.GenerateOpts{
		Root:         renderRoot,
		Output:       renderOutput,
		Channel:      "dev",
		TemplateDirs: []fs.FS{os.DirFS(templateDir)},
		DataDirs:     []fs.FS{os.DirFS(dataDir)},
	})
	require.NoError(t, err)

	renderedMarkdown, err := os.ReadFile(filepath.Join(renderDir, "out", "README.md"))
	require.NoError(t, err)

	require.Equal(t, generatedManifest.Markdown, string(renderedMarkdown))
	require.Contains(t, string(renderedMarkdown), "Note: root")
}

func TestRenderChallengeOutputsMarkdownOnly(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	templateDir := t.TempDir()
	dataDir := t.TempDir()

	writeTestFile(
		t,
		filepath.Join(templateDir, "shared.md"),
		`{{- define "shared" -}}Shared: {{ . }}{{- end -}}`,
	)
	writeTestFile(t, filepath.Join(dataDir, "note.md"), "global")
	writeTestFile(t, filepath.Join(rootDir, "data", "note.md"), "root")
	writeTestFile(t, filepath.Join(rootDir, "solution", "steps.sh"), `# @block: fix
echo fixed
# @endblock: fix
`)
	writeTestFile(t, filepath.Join(rootDir, "manifest.yaml"), `kind: challenge
name: demo-challenge-dev
title: Demo Challenge
description: Demo challenge
createdAt: 2026-01-01
playground:
  name: demo-playground
`)
	writeTestFile(t, filepath.Join(rootDir, "index.md"), `# {{ .Manifest.Title }}

Name: {{ .Name }}
Note: {{ .Extra.note }}
{{ template "shared" .Channel }}
`)
	writeTestFile(
		t,
		filepath.Join(rootDir, "solution.md"),
		"```sh\n{{ readFileBlock \"solution/steps.sh\" \"fix\" }}\n```\n",
	)

	root, err := os.OpenRoot(rootDir)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(rootDir, "out"), 0o755)
	require.NoError(t, err)

	output, err := os.OpenRoot(filepath.Join(rootDir, "out"))
	require.NoError(t, err)

	err = labx.Render(labx.GenerateOpts{
		Root:         root,
		Output:       output,
		Channel:      "dev",
		TemplateDirs: []fs.FS{os.DirFS(templateDir)},
		DataDirs:     []fs.FS{os.DirFS(dataDir)},
	})
	require.NoError(t, err)

	indexBytes, err := os.ReadFile(filepath.Join(rootDir, "out", "index.md"))
	require.NoError(t, err)

	solutionBytes, err := os.ReadFile(filepath.Join(rootDir, "out", "solution.md"))
	require.NoError(t, err)

	require.False(t, strings.HasPrefix(string(indexBytes), "---\n"))
	require.Contains(t, string(indexBytes), "Name: demo-challenge-dev")
	require.Contains(t, string(indexBytes), "Note: root")
	require.Contains(t, string(solutionBytes), "echo fixed")
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
}
