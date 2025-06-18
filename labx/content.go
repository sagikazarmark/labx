package labx

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/go-sprout/sprout"
	sproutstrings "github.com/go-sprout/sprout/registry/strings"
	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
	"github.com/sagikazarmark/labx/pkg/sproutx"
	"github.com/samber/lo"
)

func Content(root *os.Root, channel string) error {
	manifest, err := convertContentManifest(root.FS(), channel)
	if err != nil {
		return err
	}

	// Create dist directory and root
	dist, err := root.OpenRoot("dist")
	if err != nil {
		return err
	}

	indexFile, err := dist.Create("index.md")
	if err != nil {
		return err
	}
	defer indexFile.Close()

	encoder := yaml.NewEncoder(
		newFrontMatterWriter(indexFile),
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.IndentSequence(true),
	)

	err = encoder.Encode(manifest)
	if err != nil {
		return err
	}

	if strings.ToLower(channel) == "beta" {
		_, err = io.WriteString(indexFile, betaNotice)
		if err != nil {
			return err
		}
	}

	tplFuncs := sprout.New(
		sprout.WithRegistries(
			sproutstrings.NewRegistry(),
			sproutx.NewFSRegistry(root.FS()),
			sproutx.NewStringsRegistry(),
		),
	).Build()

	// Collect candidate patterns
	patternCandidates := []string{
		"*.md",
		"templates/*.md",
	}

	// Add course-specific patterns if this is a course
	if manifest.Kind == content.KindCourse {
		patternCandidates = append(patternCandidates,
			"lessons/*/*.md",
			"modules/*/*/*.md",
		)
	}

	// Filter patterns to only include those with actual matches
	var patterns []string
	for _, pattern := range patternCandidates {
		matches, err := fs.Glob(root.FS(), pattern)
		if err != nil {
			return err
		}
		if len(matches) > 0 {
			patterns = append(patterns, pattern)
		}
	}

	tpl, err := template.New("").Funcs(tplFuncs).ParseFS(root.FS(), patterns...)
	if err != nil {
		return err
	}

	err = tpl.ExecuteTemplate(indexFile, "index.md", nil)
	if err != nil {
		return err
	}

	// Handle content-specific rendering
	switch manifest.Kind {
	case content.KindChallenge:
		err := renderChallenge(root, dist, tpl)
		if err != nil {
			return err
		}
	case content.KindCourse:
		err := renderCourse(root, dist, tpl, channel)
		if err != nil {
			return err
		}
	}

	return nil
}

func convertContentManifest(fsys fs.FS, channel string) (core.ContentManifest, error) {
	manifestFile, err := fsys.Open("manifest.yaml")
	if err != nil {
		return core.ContentManifest{}, err
	}
	defer manifestFile.Close()

	decoder := yaml.NewDecoder(manifestFile)

	var extendedManifest extended.ContentManifest

	err = decoder.Decode(&extendedManifest)
	if err != nil {
		return core.ContentManifest{}, err
	}

	if extendedManifest.Kind != content.KindTraining && extendedManifest.Kind != content.KindCourse {
		hf, err := hasFiles(fsys, extendedManifest.Kind)
		if err != nil {
			return core.ContentManifest{}, err
		}

		basePlayground, err := getPlaygroundManifest(extendedManifest.Playground.Name)
		if err != nil {
			return core.ContentManifest{}, err
		}

		if hf {
			machines := lo.Map(extendedManifest.Playground.Machines, func(machine extended.PlaygroundMachine, _ int) string {
				return machine.Name
			})

			if len(machines) == 0 {
				machines = lo.Map(basePlayground.Playground.Machines, func(machine api.PlaygroundMachine, _ int) string {
					return machine.Name
				})
			}

			const name = "init_content_files"

			extendedManifest.Tasks[name] = extended.Task{
				Machine: machines,
				Init:    true,
				User:    extended.StringList{"root"},
				Run:     createDownloadScript(extendedManifest.Kind),
			}
		}

		extendedManifest.Playground.BaseName = basePlayground.Name
		extendedManifest.Playground.Base = basePlayground.Playground

		machinesProcessor := MachinesProcessor{
			MachineProcessor: MachineProcessor{
				UserProcessor: MachineUserProcessor{
					Fsys: fsys,
				},
				DriveProcessor: MachineDriveProcessor{
					ContentKind:      extendedManifest.Kind,
					ContentName:      "",
					Channel:          channel,
					DefaultImageRepo: defaultImageRepo,
				},
				StartupFileProcessor: MachineStartupFileProcessor{
					Fsys: fsys,
				},
			},
		}

		machines, err := machinesProcessor.Process(extendedManifest.Playground.Machines)
		if err != nil {
			return core.ContentManifest{}, err
		}

		extendedManifest.Playground.Machines = machines
	}

	// Apply channel-specific title processing only for real content kinds (not lessons)
	if channel != "live" && string(extendedManifest.Kind) != "lesson" {
		extendedManifest.Title = fmt.Sprintf("%s: %s", strings.ToUpper(channel), extendedManifest.Title)
	}

	manifest := extendedManifest.Convert()

	return manifest, err
}

// renderChallenge handles challenge-specific rendering
func renderChallenge(root *os.Root, dist *os.Root, tpl *template.Template) error {
	hasSolution, err := fileExists(root.FS(), "solution.md")
	if err != nil {
		return err
	}

	if hasSolution {
		solutionFile, err := dist.Create("solution.md")
		if err != nil {
			return err
		}
		defer solutionFile.Close()

		err = tpl.ExecuteTemplate(solutionFile, "solution.md", nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// renderCourse handles rendering for both simple and modular courses
func renderCourse(root *os.Root, dist *os.Root, tpl *template.Template, channel string) error {
	fsys := root.FS()

	// Check if this is a simple course (has lessons directory)
	hasLessons, err := dirExists(fsys, "lessons")
	if err != nil {
		return err
	}

	// Check if this is a modular course (has modules directory)
	hasModules, err := dirExists(fsys, "modules")
	if err != nil {
		return err
	}

	// Validate that course doesn't have both structures
	if hasLessons && hasModules {
		return fmt.Errorf("course cannot have both 'lessons' and 'modules' directories")
	}

	if hasLessons {
		return renderSimpleCourse(root, dist, tpl, channel)
	} else if hasModules {
		return renderModularCourse(root, dist, tpl, channel)
	}

	return nil
}

// renderSimpleCourse handles simple courses with a lessons directory
func renderSimpleCourse(root *os.Root, dist *os.Root, tpl *template.Template, channel string) error {
	fsys := root.FS()

	lessons, err := fs.ReadDir(fsys, "lessons")
	if err != nil {
		return err
	}

	for _, lesson := range lessons {
		if !lesson.IsDir() {
			continue
		}

		lessonName := lesson.Name()
		lessonPath := "lessons/" + lessonName

		err = renderLesson(root, dist, tpl, lessonPath, lessonName, channel)
		if err != nil {
			return fmt.Errorf("failed to render lesson %s: %w", lessonName, err)
		}
	}

	return nil
}

// renderModularCourse handles modular courses with a modules directory
func renderModularCourse(root *os.Root, dist *os.Root, tpl *template.Template, channel string) error {
	fsys := root.FS()

	modules, err := fs.ReadDir(fsys, "modules")
	if err != nil {
		return err
	}

	for _, module := range modules {
		if !module.IsDir() {
			continue
		}

		moduleName := module.Name()
		modulePath := "modules/" + moduleName

		// Process module manifest
		err = renderModuleManifest(root, dist, modulePath, moduleName)
		if err != nil {
			return fmt.Errorf("failed to render module manifest %s: %w", moduleName, err)
		}

		// Process lessons within the module
		lessons, err := fs.ReadDir(fsys, modulePath)
		if err != nil {
			return err
		}

		for _, lesson := range lessons {
			if !lesson.IsDir() {
				continue
			}

			lessonName := lesson.Name()
			lessonPath := modulePath + "/" + lessonName
			outputPath := moduleName + "/" + lessonName

			err = renderLesson(root, dist, tpl, lessonPath, outputPath, channel)
			if err != nil {
				return fmt.Errorf("failed to render lesson %s in module %s: %w", lessonName, moduleName, err)
			}
		}
	}

	return nil
}

// renderModuleManifest processes a module's manifest.yaml and creates 00-index.md
func renderModuleManifest(root *os.Root, dist *os.Root, modulePath, moduleName string) error {
	fsys := root.FS()

	manifestPath := modulePath + "/manifest.yaml"
	manifestFile, err := fsys.Open(manifestPath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	manifestContent, err := io.ReadAll(manifestFile)
	if err != nil {
		return err
	}

	outputPath := moduleName + "/00-index.md"
	outputFile, err := dist.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	fmw := newFrontMatterWriter(outputFile)
	_, err = fmw.Write(manifestContent)
	if err != nil {
		return err
	}

	return nil
}

// renderLesson processes a lesson directory and renders its content
func renderLesson(root *os.Root, dist *os.Root, tpl *template.Template, lessonPath, outputPath, channel string) error {
	fsys := root.FS()

	// Process lesson manifest through the same pipeline as other manifests
	err := renderLessonManifest(root, dist, lessonPath, outputPath, channel)
	if err != nil {
		return err
	}

	// Process markdown files in the lesson
	lessonFiles, err := fs.ReadDir(fsys, lessonPath)
	if err != nil {
		return err
	}

	for _, file := range lessonFiles {
		if file.IsDir() {
			// Handle static directory
			if file.Name() == "static" {
				err = copyStaticFiles(root, dist, lessonPath+"/static", outputPath+"/__static__")
				if err != nil {
					return fmt.Errorf("failed to copy static files: %w", err)
				}
			}
			continue
		}

		fileName := file.Name()
		if strings.HasSuffix(fileName, ".md") && fileName != "index.md" {
			templateName := lessonPath + "/" + fileName
			outputFilePath := outputPath + "/" + fileName

			outputFile, err := dist.Create(outputFilePath)
			if err != nil {
				return fmt.Errorf("failed to create output file %s: %w", outputFilePath, err)
			}
			defer outputFile.Close()

			err = tpl.ExecuteTemplate(outputFile, templateName, nil)
			if err != nil {
				return fmt.Errorf("failed to execute template %s: %w", templateName, err)
			}
		}
	}

	return nil
}

// renderLessonManifest processes a lesson's manifest.yaml and creates index.md
func renderLessonManifest(root *os.Root, dist *os.Root, lessonPath, outputPath, channel string) error {
	// Create a sub-filesystem for the lesson directory
	lessonFS, err := fs.Sub(root.FS(), lessonPath)
	if err != nil {
		return err
	}

	// Process the lesson manifest through the core pipeline with course channel but skip title processing
	manifest, err := convertContentManifest(lessonFS, channel)
	if err != nil {
		return err
	}

	// Create the output index.md file
	outputFile, err := dist.Create(outputPath + "/index.md")
	if err != nil {
		return err
	}
	defer outputFile.Close()

	encoder := yaml.NewEncoder(
		newFrontMatterWriter(outputFile),
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.IndentSequence(true),
	)

	err = encoder.Encode(manifest)
	if err != nil {
		return err
	}

	return nil
}

// copyStaticFiles copies static files from source to destination
func copyStaticFiles(root *os.Root, dist *os.Root, sourcePath, destPath string) error {
	fsys := root.FS()

	return fs.WalkDir(fsys, sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourcePath {
			return nil
		}

		// Calculate relative path from source
		relPath := strings.TrimPrefix(path, sourcePath+"/")
		outputPath := destPath + "/" + relPath

		if d.IsDir() {
			// Directory creation is handled by root.Create when creating files
			return nil
		}

		// Copy file
		sourceFile, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		destFile, err := dist.Create(outputPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, sourceFile)
		return err
	})
}

// dirExists checks if a directory exists
func dirExists(fsys fs.FS, path string) (bool, error) {
	stat, err := fs.Stat(fsys, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return stat.IsDir(), nil
}

// frontMatterWriter automatically adds front matter delimiters on first write
type frontMatterWriter struct {
	writer     io.Writer
	firstWrite bool
}

// newFrontMatterWriter creates a new frontMatterWriter
func newFrontMatterWriter(writer io.Writer) *frontMatterWriter {
	return &frontMatterWriter{writer: writer}
}

func (w *frontMatterWriter) Write(p []byte) (n int, err error) {
	if !w.firstWrite {
		w.firstWrite = true

		// Write opening delimiter
		_, err = io.WriteString(w.writer, "---\n")
		if err != nil {
			return 0, err
		}

		// Write the content
		n, err = w.writer.Write(p)
		if err != nil {
			return n, err
		}

		// Write closing delimiter
		_, err = io.WriteString(w.writer, "---\n")
		if err != nil {
			return n, err
		}

		return n, nil
	}

	// Subsequent writes go directly to the underlying writer
	return w.writer.Write(p)
}
