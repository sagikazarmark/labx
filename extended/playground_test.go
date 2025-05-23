package extended_test

import (
	"fmt"
	"testing"

	"github.com/iximiuz/labctl/api"
	"github.com/sagikazarmark/labx/extended"
	"github.com/stretchr/testify/assert"
)

func TestPlaygroundMachine_Hostname(t *testing.T) {
	startupFile := api.MachineStartupFile{
		Path:    "/foo",
		Content: "bar",
		Mode:    "755",
		Owner:   "root:root",
		Append:  false,
	}

	const hostname = "hostname"

	extendedManifest := extended.PlaygroundMachine{
		Name:         "test",
		Hostname:     hostname,
		Users:        []api.MachineUser{},
		StartupFiles: []api.MachineStartupFile{startupFile},
	}

	expected := api.PlaygroundMachine{
		Name:      "test",
		Users:     []api.MachineUser{},
		Resources: api.MachineResources{},
		StartupFiles: []api.MachineStartupFile{
			{
				Path:    "/etc/hostname",
				Content: hostname,
				Mode:    "755",
				Owner:   "root:root",
			},
			{
				Path:    "/etc/hosts",
				Content: fmt.Sprintf("127.0.0.1       %s %s.local\n", hostname, hostname),
				Append:  true,
			},
			startupFile,
		},
	}

	assert.Equal(t, expected, extendedManifest.Convert())
}
