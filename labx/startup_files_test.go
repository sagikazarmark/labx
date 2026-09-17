package labx

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sagikazarmark/labx/core"
)

func TestStartupSourcesRejectOverlappingOutput(t *testing.T) {
	t.Parallel()
	root, err := os.OpenRoot(t.TempDir())
	require.NoError(t, err)
	defer root.Close()
	require.NoError(t, root.MkdirAll("app/out", 0o755))
	require.NoError(t, root.WriteFile("app/keep.txt", []byte("keep"), 0o644))
	files := []core.StartupFile{{Source: "__static__/app.tar.gz"}}
	require.ErrorContains(t, stageStartupSources(root, root, files), "same directory")
	output, err := root.OpenRoot("app/out")
	require.NoError(t, err)
	defer output.Close()
	require.ErrorContains(t, stageStartupSources(root, output, files), "inside startup source")
	data, err := root.ReadFile("app/keep.txt")
	require.NoError(t, err)
	require.Equal(t, "keep", string(data))
}

func TestStartupSourcesLeavePrebuiltAndRemoteReferences(t *testing.T) {
	t.Parallel()
	root, err := os.OpenRoot(t.TempDir())
	require.NoError(t, err)
	defer root.Close()
	output, err := os.OpenRoot(t.TempDir())
	require.NoError(t, err)
	defer output.Close()
	require.NoError(t, output.Mkdir("__static__", 0o755))
	require.NoError(t, output.WriteFile("__static__/prebuilt.tar", []byte("prebuilt"), 0o644))
	require.NoError(t, stageStartupSources(root, output, []core.StartupFile{
		{Source: "__static__/prebuilt.tar"},
		{Source: "https://example.com/app.tar.gz"},
	}))
	data, err := output.ReadFile("__static__/prebuilt.tar")
	require.NoError(t, err)
	require.Equal(t, "prebuilt", string(data))
}
