package labx

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sagikazarmark/labx/core"
)

// Match labctl's sibling-folder archive convention.
var startupArchiveSource = regexp.MustCompile(`^__static__/([^/]+)\.(tar\.gz|tgz|tar)$`)

func stageContentStartupSources(root, output *os.Root, manifest core.ContentManifest) error {
	files := append([]core.StartupFile(nil), manifest.Playground.StartupFiles...)
	for _, machine := range manifest.Playground.Machines {
		files = append(files, machine.StartupFiles...)
	}
	return stageStartupSources(root, output, files)
}

func stagePlaygroundStartupSources(root, output *os.Root, spec core.PlaygroundSpec) error {
	files := append([]core.StartupFile(nil), spec.StartupFiles...)
	for _, machine := range spec.Machines {
		files = append(files, machine.StartupFiles...)
	}
	return stageStartupSources(root, output, files)
}

func stageStartupSources(root, output *os.Root, files []core.StartupFile) error {
	staged := map[string]bool{}
	built := map[string]bool{}
	for _, file := range files {
		match := startupArchiveSource.FindStringSubmatch(file.Source)
		if match == nil || built[file.Source] {
			continue
		}
		folder := match[1]
		if folder == "." || folder == ".." || folder == "__static__" {
			return fmt.Errorf("invalid startup archive source folder %q", folder)
		}
		exists, err := dirExists(root.FS(), folder)
		if err != nil {
			return err
		}
		if !exists {
			continue // A prebuilt archive or remote source needs no staging.
		}
		if !staged[folder] {
			if err := stageStartupFolder(root, output, folder); err != nil {
				return fmt.Errorf("stage startup source %s: %w", folder, err)
			}
			staged[folder] = true
		}
		if err := buildStartupArchive(output, folder, file.Source, match[2] != "tar"); err != nil {
			return fmt.Errorf("build startup archive %s: %w", file.Source, err)
		}
		built[file.Source] = true
	}
	// Preserve upload exclusions for staged source folders. Archive construction
	// intentionally uses only ignore rules inside the source folder, like labctl.
	if len(staged) > 0 {
		exists, err := fileExists(root.FS(), ".labctlignore")
		if err != nil {
			return err
		}
		if exists {
			return copyStartupSourceFile(root, output, ".labctlignore")
		}
	}
	return nil
}

func stageStartupFolder(root, output *os.Root, folder string) error {
	// Avoid copying an output tree into itself, or replacing the input folder.
	source, err := filepath.EvalSymlinks(filepath.Join(root.Name(), folder))
	if err != nil {
		return err
	}
	destination, err := filepath.EvalSymlinks(output.Name())
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(source, destination)
	if err != nil {
		return err
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return fmt.Errorf("output directory is inside startup source %s", folder)
	}
	if info, err := output.Stat(folder); err == nil {
		sourceInfo, err := root.Stat(folder)
		if err != nil {
			return err
		}
		if os.SameFile(sourceInfo, info) {
			return fmt.Errorf("startup source and destination are the same directory")
		}
	}
	// Replace the generated copy so removed source files do not remain in archives.
	if err := output.RemoveAll(folder); err != nil {
		return err
	}
	return fs.WalkDir(root.FS(), folder, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return output.MkdirAll(name, 0o755)
		}
		return copyStartupSourceFile(root, output, name)
	})
}

func copyStartupSourceFile(root, output *os.Root, name string) error {
	source, err := root.Open(name)
	if err != nil {
		return err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("startup source %s is not a regular file", name)
	}
	destination, err := output.OpenFile(
		name,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		info.Mode().Perm(),
	)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
