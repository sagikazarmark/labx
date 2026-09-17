package labx_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/labx"
)

func TestContentRuntimeFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.yaml"), `kind: tutorial
title: Runtime variables
description: Test runtime declarations
createdAt: 2026-09-17
vars:
  PORT: "8080"
  OUTPUT: x(.tasks.lookup_node_01.stdout)
  URL: http://localhost:x(.vars.PORT)/
  SESSION_ID:
    shell: cat /proc/sys/kernel/random/uuid
    machine: node-01
tasks:
  lookup:
    machine: [node-01, node-02]
    user: root
    helper: true
    run: echo x(.vars.PORT)
playground:
  registryAuth: test-registry
  tabs:
    - kind: terminal
      pane: secondary
      target: node-01
playgrounds:
  linux: {}
skill-paths:
  containers: {}
`)
	writeTestFile(
		t,
		filepath.Join(dir, "index.md"),
		"# {{ .Manifest.Title }}\nDeclaration: {{ .Manifest.Vars.PORT }}\nConnect to {{ `{{ vars.URL || 'waiting...' }}` }}.\n:code-var{name=PORT default='...'}\n",
	)
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	require.NoError(t, os.Mkdir(filepath.Join(dir, "out"), 0o755))
	output, err := os.OpenRoot(filepath.Join(dir, "out"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, output.Close()) })

	opts := labx.GenerateOpts{Root: root, Output: output, Channel: "live"}
	require.NoError(t, labx.Generate(opts))
	generated, err := os.ReadFile(filepath.Join(dir, "out", "index.md"))
	require.NoError(t, err)
	parts := strings.SplitN(string(generated), "---\n", 3)
	require.Len(t, parts, 3)
	var manifest core.ContentManifest
	require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &manifest))
	require.Equal(t, "8080", manifest.Vars["PORT"])
	require.Equal(t, "x(.tasks.lookup_node_01.stdout)", manifest.Vars["OUTPUT"])
	require.Equal(t, "http://localhost:x(.vars.PORT)/", manifest.Vars["URL"])
	require.Equal(t, map[string]any{
		"shell":   "cat /proc/sys/kernel/random/uuid",
		"machine": "node-01",
	}, manifest.Vars["SESSION_ID"])
	require.Len(t, manifest.Tasks, 2)
	for _, name := range []string{"lookup_node_01", "lookup_node_02"} {
		require.True(t, manifest.Tasks[name].Helper)
		require.Equal(t, "echo x(.vars.PORT)", manifest.Tasks[name].Run)
	}
	require.Equal(t, "test-registry", manifest.Playground.RegistryAuth)
	require.Contains(t, manifest.Playgrounds, "linux")
	require.Contains(t, manifest.SkillPaths, "containers")
	require.Equal(t, "secondary", manifest.Playground.Tabs[0].Pane)
	require.Equal(t, "node-01", manifest.Playground.Tabs[0].Target)
	require.Contains(t, parts[2], "Declaration: 8080")
	require.Contains(t, parts[2], "{{ vars.URL || 'waiting...' }}")
	require.Contains(t, parts[2], ":code-var{name=PORT default='...'}")

	// Render-only reads the core manifest and must expose the same declarations.
	writeTestFile(t, filepath.Join(dir, "manifest.yaml"), parts[1])
	require.NoError(t, labx.Render(opts))
	rendered, err := os.ReadFile(filepath.Join(dir, "out", "index.md"))
	require.NoError(t, err)
	require.Equal(t, parts[2], string(rendered))
}
