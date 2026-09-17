package extended

import "github.com/sagikazarmark/labx/pkg/yamlx"

func (m *ContentManifest) UnmarshalYAML(unmarshal func(any) error) error {
	type plain ContentManifest
	return yamlx.DecodeWithMetadata(unmarshal, (*plain)(m), &m.Metadata)
}

func (s *ContentPlaygroundSpec) UnmarshalYAML(unmarshal func(any) error) error {
	type plain ContentPlaygroundSpec
	return yamlx.DecodeWithMetadata(unmarshal, (*plain)(s), &s.Metadata)
}

func (t *Task) UnmarshalYAML(unmarshal func(any) error) error {
	type plain Task
	return yamlx.DecodeWithMetadata(unmarshal, (*plain)(t), &t.Metadata)
}

func (m *PlaygroundManifest) UnmarshalYAML(unmarshal func(any) error) error {
	type plain PlaygroundManifest
	return yamlx.DecodeWithMetadata(unmarshal, (*plain)(m), &m.Metadata)
}

func (s *PlaygroundSpec) UnmarshalYAML(unmarshal func(any) error) error {
	type plain PlaygroundSpec
	return yamlx.DecodeWithMetadata(unmarshal, (*plain)(s), &s.Metadata)
}

func (m *PlaygroundMachine) UnmarshalYAML(unmarshal func(any) error) error {
	type plain PlaygroundMachine
	return yamlx.DecodeWithMetadata(unmarshal, (*plain)(m), &m.Metadata)
}

func (u *MachineUser) UnmarshalYAML(unmarshal func(any) error) error {
	type plain MachineUser
	return yamlx.DecodeWithMetadata(unmarshal, (*plain)(u), &u.Metadata)
}

func (f *MachineStartupFile) UnmarshalYAML(unmarshal func(any) error) error {
	type plain MachineStartupFile
	return yamlx.DecodeWithMetadata(unmarshal, (*plain)(f), &f.Metadata)
}

func (t *InitTask) UnmarshalYAML(unmarshal func(any) error) error {
	type plain InitTask
	return yamlx.DecodeWithMetadata(unmarshal, (*plain)(t), &t.Metadata)
}
