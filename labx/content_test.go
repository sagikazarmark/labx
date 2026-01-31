package labx_test

import (
	"io/fs"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sagikazarmark/labx/labx"
)

func TestTutorials(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	testContent(t, "tutorials")
}

func testContent(t *testing.T, kind string) {
	t.Parallel()

	challenges, err := root.OpenRoot(kind)
	require.NoError(t, err)

	testGenerate(t, challenges)
}

func testGenerate(t *testing.T, content *os.Root) {
	templates, err := root.OpenRoot("_templates")
	require.NoError(t, err)

	data, err := root.OpenRoot("_data")
	require.NoError(t, err)

	fs.WalkDir(content.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() || d.Name() == "." {
			return nil
		}

		// TODO: fixme
		if d.Name() == "running-dagger-pipelines-on-github-actions" {
			return fs.SkipDir
		}

		if slices.Contains([]string{"openbao-raft", "openbao-vault-cluster-playground", "openbao-vault-playground", "sftpgo", "dagger-playground"}, d.Name()) {
			return fs.SkipDir
		}

		t.Run(d.Name(), func(t *testing.T) {
			t.Parallel()

			root, err := content.OpenRoot(path)
			require.NoError(t, err)

			output, err := os.OpenRoot(t.TempDir())
			require.NoError(t, err)

			opts := labx.GenerateOpts{
				Root:         root,
				Output:       output,
				Channel:      "dev",
				TemplateDirs: []fs.FS{templates.FS()},
				DataDirs:     []fs.FS{data.FS()},
			}

			err = labx.Generate(opts)
			require.NoError(t, err)
		})

		return fs.SkipDir
	})
}
