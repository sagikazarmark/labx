package labx

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/sagikazarmark/labx/extended"
	"github.com/samber/lo"
)

type PlaygroundProcessor struct {
	Fsys fs.FS

	Channel string

	MachinesProcessor MachinesProcessor
}

func (p PlaygroundProcessor) Process(playground extended.PlaygroundManifest) (extended.PlaygroundManifest, error) {
	if p.Channel != "live" {
		playground.Title = fmt.Sprintf("%s: %s", strings.ToUpper(p.Channel), playground.Title)
	}

	channel, ok := playground.Channels[p.Channel]
	if !ok {
		return extended.PlaygroundManifest{}, errors.New("missing channel data: " + p.Channel)
	}

	playground.Name = channel.Name

	if channel.Public {
		playground.Playground.AccessControl = api.PlaygroundAccessControl{
			CanList:  []string{"anyone"},
			CanRead:  []string{"anyone"},
			CanStart: []string{"anyone"},
		}
	}

	machines, err := p.MachinesProcessor.Process(playground.Playground.Machines)
	if err != nil {
		return extended.PlaygroundManifest{}, err
	}

	playground.Playground.Machines = machines

	hf, err := hasFiles(p.Fsys, content.KindPlayground)
	if err != nil {
		return extended.PlaygroundManifest{}, err
	}

	if hf {
		machines := lo.Map(playground.Playground.Machines, func(machine extended.PlaygroundMachine, _ int) string {
			return machine.Name
		})

		if len(machines) == 0 {
			machines = lo.Map(playground.Playground.Base.Machines, func(machine api.PlaygroundMachine, _ int) string {
				return machine.Name
			})
		}

		const name = "init_files"

		playground.Playground.InitTasks[name] = extended.InitTask{
			Name:    name,
			Machine: machines,
			Init:    true,
			User:    extended.StringList{"root"},
			Run:     createDownloadScript(content.KindPlayground),
		}
	}

	return playground, nil
}

type MachinesProcessor struct {
	MachineProcessor MachineProcessor
}

func (p MachinesProcessor) Process(machines []extended.PlaygroundMachine) ([]extended.PlaygroundMachine, error) {
	machineProcessor := p.MachineProcessor

	// Set a default drive size to make sure we don't exceed the limit
	// This is best effort: you should set the size instead
	if machineProcessor.DriveProcessor.DefaultSize == "" {
		if len(machines) > 3 {
			machineProcessor.DriveProcessor.DefaultSize = "30GiB"
		}
	}

	for i, machine := range machines {
		machine, err := machineProcessor.Process(machine)
		if err != nil {
			return nil, fmt.Errorf("processing machine %s: %w", machine.Name, err)
		}

		machines[i] = machine
	}

	return machines, nil
}

type MachineProcessor struct {
	StartupFileProcessor MachineStartupFileProcessor
	DriveProcessor       MachineDriveProcessor
}

func (p MachineProcessor) Process(machine extended.PlaygroundMachine) (extended.PlaygroundMachine, error) {
	for i, startupFile := range machine.StartupFiles {
		startupFile, err := p.StartupFileProcessor.Process(startupFile)
		if err != nil {
			return extended.PlaygroundMachine{}, fmt.Errorf("processing startup file %d: %w", i, err)
		}

		machine.StartupFiles[i] = startupFile
	}

	for i, drive := range machine.Drives {
		drive, err := p.DriveProcessor.Process(drive)
		if err != nil {
			return extended.PlaygroundMachine{}, fmt.Errorf("processing drive %d: %w", i, err)
		}

		machine.Drives[i] = drive
	}

	return machine, nil
}

type MachineStartupFileProcessor struct {
	Fsys fs.FS

	DefaultOwner string
	DefaultMode  string
}

func (p MachineStartupFileProcessor) Process(startupFile extended.MachineStartupFile) (extended.MachineStartupFile, error) {
	if startupFile.FromFile != "" {
		contentFile, err := p.Fsys.Open(startupFile.FromFile)
		if err != nil {
			return extended.MachineStartupFile{}, err
		}

		content, err := io.ReadAll(contentFile)
		if err != nil {
			return extended.MachineStartupFile{}, err
		}

		startupFile.Content = string(content)
	}

	if startupFile.Owner == "" {
		startupFile.Owner = p.DefaultOwner
	}

	if startupFile.Mode == "" {
		startupFile.Mode = p.DefaultMode
	}

	return startupFile, nil
}

type MachineDriveProcessor struct {
	// TODO: this won' work for course modules and lessons
	ContentKind content.ContentKind
	ContentName string
	Channel     string

	// Use this image repository if the image is not specified.
	DefaultImageRepo string

	// Use this size if the size is missing from the drive.
	DefaultSize string
}

func (p MachineDriveProcessor) Process(drive api.MachineDrive) (api.MachineDrive, error) {
	source, err := p.processSource(drive.Source)
	if err != nil {
		return api.MachineDrive{}, err
	}

	drive.Source = source

	if drive.Size == "" {
		drive.Size = p.DefaultSize
	}

	return drive, nil
}

func (p MachineDriveProcessor) processSource(source string) (string, error) {
	// Not an OCI source, stop processing
	if !strings.HasPrefix(source, "oci://") {
		return source, nil
	}

	source = strings.TrimPrefix(source, "oci://")

	// Fallback to default source
	if source == "" {
		source = fmt.Sprintf("%s/%s/%s:%s", p.DefaultImageRepo, p.ContentKind.Plural(), p.ContentName, p.Channel)
	}

	// Replace channel placeholder
	source = strings.ReplaceAll(source, "__CHANNEL__", p.Channel)

	ref, err := name.ParseReference(source)
	if err != nil {
		return "", err
	}

	// Already pinned to a digest
	if _, ok := ref.(name.Digest); ok {
		return source, nil
	}

	desc, err := remote.Get(ref)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("oci://%s@%s", ref.String(), desc.Digest.String()), nil
}
