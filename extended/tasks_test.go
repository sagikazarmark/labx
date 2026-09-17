package extended_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/stretchr/testify/require"

	"github.com/sagikazarmark/labx/extended"
)

func TestTaskDefaultTargets(t *testing.T) {
	t.Parallel()
	for _, targets := range []struct {
		name    string
		yaml    string
		names   []string
		omitted []string
	}{
		{"both omitted", "", []string{"work"}, []string{"machine", "user"}},
		{"machine omitted", "user: root\n", []string{"work"}, []string{"machine"}},
		{"user omitted", "machine: node-01\n", []string{"work"}, []string{"user"}},
		{"explicit", "machine: node-01\nuser: root\n", []string{"work"}, nil},
		{"expanded machines", "machine: [node-01, node-02]\n", []string{"work_node_01", "work_node_02"}, []string{"user"}},
		{"expanded users", "user: [root, laborant]\n", []string{"work_root", "work_laborant"}, []string{"machine"}},
	} {
		t.Run(targets.name, func(t *testing.T) {
			for _, kind := range []string{"content", "playground"} {
				t.Run(kind, func(t *testing.T) {
					var converted any
					if kind == "content" {
						var task extended.Task
						require.NoError(
							t,
							yaml.Unmarshal(
								[]byte(
									targets.yaml+"run: echo ready\nneeds: [prepare]\nfuture: {enabled: true}\n",
								),
								&task,
							),
						)
						manifest := extended.ContentManifest{Tasks: map[string]extended.Task{
							"prepare": {Run: "true"}, "work": task,
						}}
						converted = manifest.Convert().Tasks
					} else {
						var task extended.InitTask
						require.NoError(
							t,
							yaml.Unmarshal(
								[]byte(
									targets.yaml+"run: echo ready\nneeds: [prepare]\nfuture: {enabled: true}\n",
								),
								&task,
							),
						)
						converted = extended.InitTasks{
							"prepare": {Run: "true"},
							"work":    task,
						}.Convert()
					}
					data, err := yaml.Marshal(converted)
					require.NoError(t, err)
					var tasks map[string]map[string]any
					require.NoError(t, yaml.Unmarshal(data, &tasks))
					require.Len(t, tasks, len(targets.names)+1)
					for _, name := range targets.names {
						require.Contains(t, tasks, name)
						require.Equal(t, "echo ready", tasks[name]["run"])
						require.Equal(t, []any{"prepare"}, tasks[name]["needs"])
						require.Equal(t, map[string]any{"enabled": true}, tasks[name]["future"])
						for _, key := range targets.omitted {
							require.NotContains(t, tasks[name], key)
						}
					}
				})
			}
		})
	}
}

func TestContentDependenciesOnPlaygroundInitTasks(t *testing.T) {
	t.Parallel()
	manifest := extended.ContentManifest{
		Playground: extended.ContentPlaygroundSpec{
			Base: api.PlaygroundSpec{InitTasks: map[string]api.InitTask{"base": {Run: "true"}}},
			InitTasks: extended.InitTasks{
				"prepare": {
					Machine: extended.StringList{"node-01", "node-02"},
					Needs:   []string{"base"}, Run: "echo ready",
				},
			},
		},
		Tasks: map[string]extended.Task{
			"check": {
				Machine: extended.StringList{"node-01", "node-02"},
				Needs:   []string{"prepare"}, Run: "true",
			},
		},
	}
	converted := manifest.Convert()
	require.Len(t, converted.Playground.InitTasks, 2)
	require.Equal(t, []string{"base"}, converted.Playground.InitTasks["prepare_node_01"].Needs)
	require.Equal(t, []string{"prepare_node_01"}, converted.Tasks["check_node_01"].Needs)
	require.Equal(t, []string{"prepare_node_02"}, converted.Tasks["check_node_02"].Needs)
	// Conversion must not mutate shared needs slices or duplicate inherited tasks.
	require.Equal(t, converted, manifest.Convert())
	require.Equal(t, []string{"prepare"}, manifest.Tasks["check"].Needs)
}

func TestTaskMetadataWithYAMLMerge(t *testing.T) {
	t.Parallel()
	var manifest extended.ContentManifest
	require.NoError(t, yaml.Unmarshal([]byte(`tasks:
  prepare: &common
    run: echo ready
    future: {enabled: true}
  follow:
    <<: *common
    needs: [prepare]
`), &manifest))
	converted := manifest.Convert()
	require.Equal(t, "echo ready", converted.Tasks["follow"].Run)
	require.Equal(t, []string{"prepare"}, converted.Tasks["follow"].Needs)
	require.Equal(
		t,
		map[string]any{"future": map[string]any{"enabled": true}},
		converted.Tasks["follow"].Metadata,
	)
}
