package labx_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/sagikazarmark/labx/labx"
)

func TestNestedMetadataAlongsideTransformations(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"tutorial", "playground"} {
		t.Run(kind, func(t *testing.T) {
			dir, opts := compatibilityRoots(t)
			writeTestFile(t, filepath.Join(dir, "manifest.yaml"), `kind: `+kind+`
title: Nested metadata
channels:
  live:
    name: nested-metadata
playground:
  futureSpec:
    enabled: true
  initConditions:
    values:
      - key: flavor
        default: vanilla
  portForwards:
    - kind: local
      machine: node-01
      localPort: 18080
      remotePort: 8080
  initTasks:
    prepare:
      run: echo ready
      futureTask: {retries: 3}
    configure:
      machine: [node-01, node-02]
      needs: [prepare]
      run: echo configured
      futureTask: {retries: 4}
  startupFiles:
    - path: /etc/global
      fromFile: message.txt
      futureFile: {encoding: utf8}
  machines:
    - name: node-01
      hostname: custom-node
      futureMachine: true
      users:
        - name: laborant
          welcomeFile: message.txt
          futureUser: [one, two]
      startupFiles:
        - path: /etc/local
          fromFile: message.txt
          futureFile: {encoding: ascii}
`)
			writeTestFile(t, filepath.Join(dir, "message.txt"), "hello\n")
			body := "# {{ .Manifest.Title }}\nFuture: {{ .Manifest.Playground.Metadata.futureSpec.enabled }}\n"
			writeTestFile(t, filepath.Join(dir, "index.md"), body)
			writeTestFile(t, filepath.Join(dir, "README.md"), body)
			require.NoError(t, labx.Generate(opts))
			var manifestYAML []byte
			if kind == "playground" {
				var err error
				manifestYAML, err = opts.Output.ReadFile("manifest.yaml")
				require.NoError(t, err)
			} else {
				index, err := opts.Output.ReadFile("index.md")
				require.NoError(t, err)
				manifestYAML = []byte(strings.SplitN(string(index), "---\n", 3)[1])
			}
			var manifest map[string]any
			require.NoError(t, yaml.Unmarshal(manifestYAML, &manifest))
			playground := yamlMap(t, manifest["playground"])
			require.Equal(t, map[string]any{"enabled": true}, playground["futureSpec"])
			conditions := yamlMap(t, playground["initConditions"])["values"].([]any)
			require.Equal(t, "vanilla", yamlMap(t, conditions[0])["default"])
			forwards := playground["portForwards"].([]any)
			require.EqualValues(t, 18080, yamlMap(t, forwards[0])["localPort"])
			initTasks := yamlMap(t, playground["initTasks"])
			require.Len(t, initTasks, 3)
			prepare := yamlMap(t, initTasks["prepare"])
			require.NotContains(t, prepare, "machine")
			require.NotContains(t, prepare, "user")
			require.EqualValues(t, 3, yamlMap(t, prepare["futureTask"])["retries"])
			for _, name := range []string{"configure_node_01", "configure_node_02"} {
				task := yamlMap(t, initTasks[name])
				require.Equal(t, []any{"prepare"}, task["needs"])
				require.IsType(t, "", task["machine"])
				require.NotContains(t, task, "user")
				require.EqualValues(t, 4, yamlMap(t, task["futureTask"])["retries"])
			}
			files := playground["startupFiles"].([]any)
			file := yamlMap(t, files[0])
			require.Equal(t, "hello\n", file["content"])
			require.NotContains(t, file, "fromFile")
			require.Equal(t, "utf8", yamlMap(t, file["futureFile"])["encoding"])
			machines := playground["machines"].([]any)
			machine := yamlMap(t, machines[0])
			require.Equal(t, true, machine["futureMachine"])
			require.NotContains(t, machine, "hostname")
			users := machine["users"].([]any)
			user := yamlMap(t, users[0])
			require.Equal(t, []any{"one", "two"}, user["futureUser"])
			require.Equal(t, "hello\n", user["welcome"])
			require.NotContains(t, user, "welcomeFile")
			files = machine["startupFiles"].([]any)
			require.Len(t, files, 3) // Generated hostname/hosts plus the author's file.
			require.Equal(t, "custom-node", yamlMap(t, files[0])["content"])
			file = yamlMap(t, files[2])
			require.Equal(t, "ascii", yamlMap(t, file["futureFile"])["encoding"])
			require.Equal(t, "hello\n", file["content"])
			require.NotContains(t, file, "fromFile")

			// Render-only reads the same lossless core models.
			require.NoError(t, opts.Root.WriteFile("manifest.yaml", manifestYAML, 0o644))
			require.NoError(t, labx.Render(opts))
			output := "index.md"
			if kind == "playground" {
				output = "README.md"
			}
			rendered, err := opts.Output.ReadFile(output)
			require.NoError(t, err)
			require.Contains(t, string(rendered), "Future: true")
		})
	}
}

func yamlMap(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	require.True(t, ok, "expected YAML mapping, got %T", value)
	return result
}
