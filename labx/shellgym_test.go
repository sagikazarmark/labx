package labx_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
	"github.com/sagikazarmark/labx/labx"
)

func TestShellGymFixture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("labctl stub uses a POSIX shell")
	}
	dir, opts := compatibilityRoots(t)
	fixture := os.DirFS("testdata/shell-gym")
	require.NoError(
		t,
		fs.WalkDir(fixture, ".", func(name string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			data, err := fs.ReadFile(fixture, name)
			if err != nil {
				return err
			}
			writeTestFile(t, filepath.Join(dir, name), string(data))
			return nil
		}),
	)
	// Exercise named-playground loading without provisioning a remote session.
	bin := t.TempDir()
	stub := filepath.Join(bin, "labctl")
	writeTestFile(t, stub, `#!/bin/sh
[ "$1 $2 $3" = "playground manifest ubuntu-26-04" ] || exit 1
cat <<'YAML'
kind: playground
name: ubuntu-26-04
playground:
  machines:
    - name: dev-machine
      users:
        - name: laborant
          default: true
YAML
`)
	require.NoError(t, os.Chmod(stub, 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	require.NoError(t, labx.Generate(opts))
	index, err := opts.Output.ReadFile("index.md")
	require.NoError(t, err)
	parts := strings.SplitN(string(index), "---\n", 3)
	require.Len(t, parts, 3)
	var manifest core.ContentManifest
	require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &manifest))
	require.NotNil(t, manifest.ShellGym)
	require.Equal(t, "latest", *manifest.ShellGym.Version)
	require.Equal(t, "/opt/shellgym/path", *manifest.ShellGym.Path)
	require.Equal(t, 63636, *manifest.ShellGym.Port)
	require.NotNil(t, manifest.Repetition)
	require.Equal(t, []int{1, 3, 7, 14, 30}, *manifest.Repetition)
	require.NotContains(t, manifest.Metadata, "shellgym")
	require.NotContains(t, manifest.Metadata, "repetition")
	require.Equal(t, *manifest.ShellGym.Path, manifest.Playground.StartupFiles[0].Path)
	require.True(t, manifest.Playground.StartupFiles[0].Extract)
	require.Contains(t, manifest.Playground.InitTasks, "prepare")

	archive, err := opts.Output.ReadFile("__static__/path.tar.gz")
	require.NoError(t, err)
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	require.NoError(t, err)
	defer gz.Close()
	tr := tar.NewReader(gz)
	archived := map[string][]byte{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		archived[header.Name], err = io.ReadAll(tr)
		require.NoError(t, err)
	}
	count := 0
	require.NoError(
		t,
		fs.WalkDir(fixture, "path", func(name string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			expected, err := fs.ReadFile(fixture, name)
			require.NoError(t, err)
			staged, err := opts.Output.ReadFile(name)
			require.NoError(t, err)
			require.Equal(t, expected, staged)
			require.Equal(t, expected, archived[strings.TrimPrefix(name, "path/")])
			count++
			return nil
		}),
	)
	require.Len(t, archived, count)
	_, err = opts.Output.Stat(".labctlignore")
	require.ErrorIs(t, err, os.ErrNotExist) // Raw path files remain uploadable/searchable.

	require.NoError(t, opts.Root.WriteFile("manifest.yaml", []byte(parts[1]), 0o644))
	require.NoError(t, labx.Render(opts))
	rendered, err := opts.Output.ReadFile("index.md")
	require.NoError(t, err)
	require.Equal(t, parts[2], string(rendered))
}

func TestShellGymOmittedSettings(t *testing.T) {
	t.Parallel()
	for _, source := range []string{
		"kind: shell-gym\n",
		"kind: shell-gym\nshellgym: {}\n",
		"kind: shell-gym\nshellgym: {port: 0, version: '', future: true}\n",
		"kind: shell-gym\nrepetition: []\n",
	} {
		var manifest extended.ContentManifest
		require.NoError(t, yaml.Unmarshal([]byte(source), &manifest))
		data, err := yaml.Marshal(manifest.Convert())
		require.NoError(t, err)
		var result, original map[string]any
		require.NoError(t, yaml.Unmarshal(data, &result))
		require.NoError(t, yaml.Unmarshal([]byte(source), &original))
		require.Equal(t, original["shellgym"], result["shellgym"])
		require.Equal(t, original["repetition"], result["repetition"])
	}
}
