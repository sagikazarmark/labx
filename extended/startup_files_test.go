package extended_test

import (
	"testing"

	"github.com/iximiuz/labctl/api"
	"github.com/stretchr/testify/require"

	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
)

func TestContentStartupFilesInheritance(t *testing.T) {
	t.Parallel()
	parent := api.StartupFile{Path: "/etc/parent", Content: "parent", Machines: []string{"node-01"}}
	spec := extended.ContentPlaygroundSpec{
		Base: api.PlaygroundSpec{StartupFiles: []api.StartupFile{parent}},
		StartupFiles: extended.MachineStartupFiles{
			{Path: "/opt/app", Source: "__static__/app.tar", Extract: true},
		},
	}
	converted := spec.Convert()
	require.Empty(t, converted.Machines)
	require.Equal(t, []core.StartupFile{
		core.StartupFilesFromAPI([]api.StartupFile{parent})[0],
		{Path: "/opt/app", Source: "__static__/app.tar", Extract: true},
	}, converted.StartupFiles)
	require.Len(t, spec.Base.StartupFiles, 1)
	require.Equal(t, converted, spec.Convert())
}
