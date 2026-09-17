package labx_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/stretchr/testify/require"

	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/labx"
)

func compatibilityRoots(t *testing.T) (string, labx.GenerateOpts) {
	t.Helper()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	require.NoError(t, os.Mkdir(filepath.Join(dir, "out"), 0o755))
	output, err := root.OpenRoot("out")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, output.Close()) })
	return dir, labx.GenerateOpts{Root: root, Output: output, Channel: "live"}
}

func TestStartupFilesGeneration(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"playground", "tutorial"} {
		t.Run(kind, func(t *testing.T) {
			dir, opts := compatibilityRoots(t)
			writeTestFile(t, filepath.Join(dir, "manifest.yaml"), `kind: `+kind+`
title: Startup files
channels:
  live:
    name: startup-files
playground:
  startupFiles:
    - path: /opt/app
      source: __static__/app.tar.gz
      extract: true
      machines: [node-01]
    - path: /etc/message
      fromFile: message.txt
    - path: /opt/app-copy
      source: __static__/app.tar
      extract: true
  machines:
    - name: node-01
      startupFiles:
        - path: /opt/data
          source: __static__/data.tgz
          extract: true
        - path: /etc/local
          fromFile: message.txt
  portForwards:
    - machine: node-01
      kind: local
      localPort: 18080
      remotePort: 8080
`)
			writeTestFile(t, filepath.Join(dir, "index.md"), "Startup files")
			writeTestFile(t, filepath.Join(dir, "message.txt"), "hello\n")
			writeTestFile(t, filepath.Join(dir, ".labctlignore"), "app/\ndata/\n")
			writeTestFile(t, filepath.Join(dir, "app", "run.sh"), "#!/bin/sh\necho hello\n")
			require.NoError(t, os.Chmod(filepath.Join(dir, "app", "run.sh"), 0o755))
			writeTestFile(t, filepath.Join(dir, "app", ".labctlignore"), "*.log\n")
			writeTestFile(t, filepath.Join(dir, "app", "debug.log"), "ignored")
			writeTestFile(t, filepath.Join(dir, "app", "nested", ".labctlignore"), "secret.txt\n")
			writeTestFile(t, filepath.Join(dir, "app", "nested", "secret.txt"), "ignored")
			writeTestFile(t, filepath.Join(dir, "app", "nested", "keep.txt"), "keep")
			writeTestFile(t, filepath.Join(dir, "data", "nested", "value.txt"), "42")
			require.NoError(t, labx.Generate(opts))
			var top, machine []core.StartupFile
			if kind == "playground" {
				data, err := opts.Output.ReadFile("manifest.yaml")
				require.NoError(t, err)
				var manifest api.PlaygroundManifest
				require.NoError(t, yaml.Unmarshal(data, &manifest))
				top = core.StartupFilesFromAPI(manifest.Playground.StartupFiles)
				machine = core.StartupFilesFromAPI(manifest.Playground.Machines[0].StartupFiles)
				require.Len(t, manifest.Playground.PortForwards, 1)
				require.Equal(t, "node-01", manifest.Playground.PortForwards[0].Machine)
				require.Equal(t, 18080, manifest.Playground.PortForwards[0].LocalPort)
				require.Equal(t, 8080, manifest.Playground.PortForwards[0].RemotePort)
			} else {
				data, err := opts.Output.ReadFile("index.md")
				require.NoError(t, err)
				var manifest core.ContentManifest
				require.NoError(
					t,
					yaml.Unmarshal([]byte(strings.SplitN(string(data), "---\n", 3)[1]), &manifest),
				)
				top = manifest.Playground.StartupFiles
				machine = manifest.Playground.Machines[0].StartupFiles
			}
			require.Len(t, top, 3)
			require.Equal(t, "__static__/app.tar.gz", top[0].Source)
			require.True(t, top[0].Extract)
			require.Equal(t, []string{"node-01"}, top[0].Machines)
			require.Equal(t, "hello\n", top[1].Content)
			require.Equal(t, "__static__/data.tgz", machine[0].Source)
			require.True(t, machine[0].Extract)
			require.Equal(t, "hello\n", machine[1].Content)
			info, err := opts.Output.Stat("app/run.sh")
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
			data, err := opts.Output.ReadFile("data/nested/value.txt")
			require.NoError(t, err)
			require.Equal(t, "42", string(data))
			data, err = opts.Output.ReadFile("app/.labctlignore")
			require.NoError(t, err)
			require.Equal(t, "*.log\n", string(data))
			data, err = opts.Output.ReadFile(".labctlignore")
			require.NoError(t, err)
			require.Equal(t, "app/\ndata/\n", string(data))
			archive, err := opts.Output.ReadFile("__static__/app.tar.gz")
			require.NoError(t, err)
			gz, err := gzip.NewReader(bytes.NewReader(archive))
			require.NoError(t, err)
			tr := tar.NewReader(gz)
			entries := map[string]string{}
			for {
				header, err := tr.Next()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				data, err := io.ReadAll(tr)
				require.NoError(t, err)
				entries[header.Name] = string(data)
				if header.Name == "run.sh" {
					require.Equal(t, int64(0o755), header.Mode)
				}
			}
			require.NoError(t, gz.Close())
			require.Equal(t, map[string]string{
				"run.sh":          "#!/bin/sh\necho hello\n",
				"nested/keep.txt": "keep",
			}, entries)
			require.NoError(t, labx.Generate(opts))
			rebuilt, err := opts.Output.ReadFile("__static__/app.tar.gz")
			require.NoError(t, err)
			require.Equal(t, archive, rebuilt)
			plain, err := opts.Output.ReadFile("__static__/app.tar")
			require.NoError(t, err)
			_, err = tar.NewReader(bytes.NewReader(plain)).Next()
			require.NoError(t, err)
			// Rebuilding must not retain deleted files in the staged archive source.
			require.NoError(t, opts.Root.Remove("app/run.sh"))
			require.NoError(t, labx.Generate(opts))
			_, err = opts.Output.Stat("app/run.sh")
			require.ErrorIs(t, err, os.ErrNotExist)
			var coreManifest []byte
			if kind == "playground" {
				coreManifest, err = opts.Output.ReadFile("manifest.yaml")
				require.NoError(t, err)
			} else {
				generated, err := opts.Output.ReadFile("index.md")
				require.NoError(t, err)
				coreManifest = []byte(strings.SplitN(string(generated), "---\n", 3)[1])
			}
			require.NoError(t, opts.Root.WriteFile("manifest.yaml", coreManifest, 0o644))
			require.NoError(t, labx.Render(opts))
		})
	}
}

func TestAdditionalContentKinds(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"skill-path", "blog-post", "shell-gym", "roadmap", "vendor"} {
		t.Run(kind, func(t *testing.T) {
			dir, opts := compatibilityRoots(t)
			writeTestFile(t, filepath.Join(dir, "manifest.yaml"), `kind: `+kind+`
title: New content
difficulties: [easy, medium]
upstreamMetadata:
  nested:
    - value: preserved
`)
			writeTestFile(t, filepath.Join(dir, "index.md"), "# {{ .Manifest.Title }}\n")
			if kind == "skill-path" {
				writeTestFile(
					t,
					filepath.Join(dir, "unit-10.md"),
					"---\nkind: unit\nname: intro\n---\n{{ .Manifest.Title }}\n",
				)
				writeTestFile(
					t,
					filepath.Join(dir, "units", "unit-20.md"),
					"---\nkind: unit\n---\n{{ .Manifest.Title }}\n",
				)
			}
			require.NoError(t, labx.Generate(opts))
			generated, err := opts.Output.ReadFile("index.md")
			require.NoError(t, err)
			parts := strings.SplitN(string(generated), "---\n", 3)
			require.Len(t, parts, 3)
			var front map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &front))
			require.Equal(t, kind, front["kind"])
			require.Contains(t, front, "upstreamMetadata")
			require.Equal(t, map[string]any{
				"nested": []any{map[string]any{"value": "preserved"}},
			}, front["upstreamMetadata"])
			require.Equal(t, []any{"easy", "medium"}, front["difficulties"])
			writeTestFile(t, filepath.Join(dir, "manifest.yaml"), parts[1])
			require.NoError(t, labx.Render(opts))
			rendered, err := opts.Output.ReadFile("index.md")
			require.NoError(t, err)
			require.Equal(t, parts[2], string(rendered))
			if kind == "skill-path" {
				for _, name := range []string{"unit-10.md", "unit-20.md"} {
					data, err := opts.Output.ReadFile(name)
					require.NoError(t, err)
					require.Contains(t, string(data), "kind: unit")
					require.Contains(t, string(data), "New content")
				}
			}
		})
	}
}

func TestCourseLessonStartupArchive(t *testing.T) {
	t.Parallel()
	dir, opts := compatibilityRoots(t)
	writeTestFile(t, filepath.Join(dir, "manifest.yaml"), "kind: course\ntitle: Course\n")
	writeTestFile(t, filepath.Join(dir, "index.md"), "Course")
	writeTestFile(t, filepath.Join(dir, "lessons", "intro", "manifest.yaml"), `kind: lesson
title: Introduction
playground:
  startupFiles:
    - path: /opt/app
      source: __static__/app.tar
      extract: true
`)
	writeTestFile(t, filepath.Join(dir, "lessons", "intro", "app", "hello.txt"), "hello")
	require.NoError(t, labx.Generate(opts))
	data, err := opts.Output.ReadFile("intro/__static__/app.tar")
	require.NoError(t, err)
	tr := tar.NewReader(bytes.NewReader(data))
	header, err := tr.Next()
	require.NoError(t, err)
	require.Equal(t, "hello.txt", header.Name)
	data, err = io.ReadAll(tr)
	require.NoError(t, err)
	require.Equal(t, "hello", string(data))
	_, err = opts.Output.Stat("intro/app/hello.txt")
	require.NoError(t, err)
}
