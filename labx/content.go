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
	sprouttime "github.com/go-sprout/sprout/registry/time"
	"github.com/goccy/go-yaml"
	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"
	"github.com/samber/lo"

	"github.com/sagikazarmark/labx/core"
	"github.com/sagikazarmark/labx/extended"
	"github.com/sagikazarmark/labx/pkg/sproutx"
)

func Content(root *os.Root, output *os.Root, channel string) error {
	manifest, err := convertContentManifest(root.FS(), channel)
	if err != nil {
		return err
	}

	indexFile, err := output.Create("index.md")
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

	tpl, err := createContentTemplate(root.FS())
	if err != nil {
		return err
	}

	err = tpl.ExecuteTemplate(indexFile, "index.md", nil)
	if err != nil {
		return err
	}

	// Copy static files if they exist at the root level
	hasStatic, err := dirExists(root.FS(), "static")
	if err != nil {
		return err
	}

	if hasStatic {
		err = copyStaticFiles(root, output, "static", "__static__")
		if err != nil {
			return fmt.Errorf("copy static files: %w", err)
		}
	}

	// Handle content-specific rendering
	switch manifest.Kind {
	case content.KindChallenge:
		err := renderChallenge(root, output, tpl)
		if err != nil {
			return err
		}
	case content.KindCourse:
		err := renderCourse(root, output, channel)
		if err != nil {
			return err
		}
	case content.KindTraining:
		err := renderTraining(root, output, tpl)
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

	if extendedManifest.Playground.Name != "" {
		hf, err := hasFiles(fsys, extendedManifest.Kind)
		if err != nil {
			return core.ContentManifest{}, err
		}

		basePlayground, err := getPlaygroundManifest(extendedManifest.Playground.Name)
		if err != nil {
			return core.ContentManifest{}, err
		}

		if hf {
			machines := lo.Map(
				extendedManifest.Playground.Machines,
				func(machine extended.PlaygroundMachine, _ int) string {
					return machine.Name
				},
			)

			if len(machines) == 0 {
				machines = lo.Map(
					basePlayground.Playground.Machines,
					func(machine api.PlaygroundMachine, _ int) string {
						return machine.Name
					},
				)
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
		extendedManifest.Title = fmt.Sprintf(
			"%s: %s",
			strings.ToUpper(channel),
			extendedManifest.Title,
		)
	}

	manifest := extendedManifest.Convert()

	return manifest, err
}

// renderChallenge handles challenge-specific rendering
func renderChallenge(root *os.Root, output *os.Root, tpl *template.Template) error {
	hasSolution, err := fileExists(root.FS(), "solution.md")
	if err != nil {
		return err
	}

	if hasSolution {
		solutionFile, err := output.Create("solution.md")
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

// renderTraining handles training-specific rendering
func renderTraining(root *os.Root, output *os.Root, tpl *template.Template) error {
	fsys := root.FS()

	// Process program.md if it exists
	hasProgramFile, err := fileExists(fsys, "program.md")
	if err != nil {
		return err
	}

	if hasProgramFile {
		programFile, err := output.Create("program.md")
		if err != nil {
			return err
		}
		defer programFile.Close()

		err = tpl.ExecuteTemplate(programFile, "program.md", nil)
		if err != nil {
			return err
		}
	}

	// Process units directory if it exists
	hasUnits, err := dirExists(fsys, "units")
	if err != nil {
		return err
	}

	if hasUnits {
		units, err := fs.ReadDir(fsys, "units")
		if err != nil {
			return err
		}

		for _, unit := range units {
			if unit.IsDir() {
				continue
			}

			unitName := unit.Name()
			if !strings.HasSuffix(unitName, ".md") {
				continue
			}

			err = renderUnit(root, output, "units", unitName)
			if err != nil {
				return fmt.Errorf("render unit %s: %w", unitName, err)
			}
		}
	}

	return nil
}

// renderCourse handles rendering for both simple and modular courses
func renderCourse(root *os.Root, output *os.Root, channel string) error {
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
		return renderSimpleCourse(root, output, channel)
	} else if hasModules {
		return renderModularCourse(root, output, channel)
	}

	return nil
}

// renderSimpleCourse handles simple courses with a lessons directory
func renderSimpleCourse(root *os.Root, output *os.Root, channel string) error {
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

		err = renderLesson(root, output, lessonPath, lessonName, channel)
		if err != nil {
			return fmt.Errorf("render lesson %s: %w", lessonName, err)
		}
	}

	return nil
}

// renderModularCourse handles modular courses with a modules directory
func renderModularCourse(root *os.Root, output *os.Root, channel string) error {
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
		err = renderModuleManifest(root, output, modulePath, moduleName)
		if err != nil {
			return fmt.Errorf("render module manifest %s: %w", moduleName, err)
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

			err = renderLesson(root, output, lessonPath, outputPath, channel)
			if err != nil {
				return fmt.Errorf(
					"render lesson %s in module %s: %w",
					lessonName,
					moduleName,
					err,
				)
			}
		}
	}

	return nil
}

// renderModuleManifest processes a module's manifest.yaml and creates 00-index.md
func renderModuleManifest(root *os.Root, output *os.Root, modulePath, moduleName string) error {
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

	// Create module directory first
	err = output.Mkdir(moduleName, 0o755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	outputFile, err := output.Create(outputPath)
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
func renderLesson(root *os.Root, output *os.Root, lessonPath, outputPath, channel string) error {
	fsys := root.FS()

	// Create lesson directory first
	err := output.Mkdir(outputPath, 0o755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Process lesson manifest through the same pipeline as other manifests
	err = renderLessonManifest(root, output, lessonPath, outputPath, channel)
	if err != nil {
		return err
	}

	// Create a sub-filesystem constrained to the lesson directory
	lessonFS, err := fs.Sub(fsys, lessonPath)
	if err != nil {
		return fmt.Errorf("create lesson sub-filesystem: %w", err)
	}

	// Create lesson-specific template instance with access to course-level templates
	tpl, err := createLessonTemplate(root.FS(), lessonFS)
	if err != nil {
		return fmt.Errorf("create lesson template: %w", err)
	}

	// Find files in the lesson directory
	lessonFiles, err := fs.ReadDir(lessonFS, ".")
	if err != nil {
		return err
	}

	for _, file := range lessonFiles {
		if file.IsDir() {
			// Handle static directory
			if file.Name() == "static" {
				err = copyStaticFiles(root, output, lessonPath+"/static", outputPath+"/__static__")
				if err != nil {
					return fmt.Errorf("copy static files: %w", err)
				}
			}
			continue
		}

		fileName := file.Name()
		if strings.HasSuffix(fileName, ".md") && fileName != "index.md" {
			outputFilePath := outputPath + "/" + fileName

			outputFile, err := output.Create(outputFilePath)
			if err != nil {
				return fmt.Errorf("create output file %s: %w", outputFilePath, err)
			}
			defer outputFile.Close()

			err = tpl.ExecuteTemplate(outputFile, fileName, nil)
			if err != nil {
				return fmt.Errorf("execute template %s: %w", fileName, err)
			}
		}
	}

	return nil
}

// renderLessonManifest processes a lesson's manifest.yaml and creates index.md
func renderLessonManifest(
	root *os.Root,
	output *os.Root,
	lessonPath, outputPath, channel string,
) error {
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

	// Create the output 00-index.md file
	outputFile, err := output.Create(outputPath + "/00-index.md")
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
func copyStaticFiles(root *os.Root, output *os.Root, sourcePath, destPath string) error {
	fsys := root.FS()

	// Create the parent static directory first
	err := output.Mkdir(destPath, 0o755)
	if err != nil && !os.IsExist(err) {
		return err
	}

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
			// Create directory in destination
			err = output.Mkdir(outputPath, 0o755)
			if err != nil && !os.IsExist(err) {
				return err
			}
			return nil
		}

		// Copy file
		sourceFile, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		destFile, err := output.Create(outputPath)
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

// createTemplateFuncs creates template functions for the given filesystem
func createTemplateFuncs(fsys fs.FS) template.FuncMap {
	return sprout.New(
		sprout.WithRegistries(
			sproutstrings.NewRegistry(),
			sproutx.NewFSRegistry(fsys),
			sproutx.NewStringsRegistry(),
			sprouttime.NewRegistry(),
		),
	).Build()
}

// parseTemplatePatterns parses template patterns from a filesystem into a template
func parseTemplatePatterns(
	tpl *template.Template,
	fsys fs.FS,
	patterns []string,
) (*template.Template, error) {
	for _, pattern := range patterns {
		matches, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, err
		}

		if len(matches) == 0 {
			continue
		}

		tpl, err = tpl.ParseFS(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("parse templates with pattern %s: %w", pattern, err)
		}
	}
	return tpl, nil
}

// createContentTemplate creates a template instance for content-level rendering
func createContentTemplate(fsys fs.FS) (*template.Template, error) {
	tplFuncs := createTemplateFuncs(fsys)
	tpl := template.New("").Funcs(tplFuncs)

	// Parse content-level patterns
	patterns := []string{
		"*.md",
		"templates/*.md",
	}

	return parseTemplatePatterns(tpl, fsys, patterns)
}

// renderUnit processes a unit file and renders its content
func renderUnit(root *os.Root, output *os.Root, unitPath, unitName string) error {
	fsys := root.FS()

	// Create a sub-filesystem constrained to the units directory
	unitsFS, err := fs.Sub(fsys, unitPath)
	if err != nil {
		return fmt.Errorf("create units sub-filesystem: %w", err)
	}

	// Create unit-specific template instance with access to training-level templates
	tpl, err := createUnitTemplate(root.FS(), unitsFS)
	if err != nil {
		return fmt.Errorf("create unit template: %w", err)
	}

	// Copy and process markdown files from units/ to the root
	outputFile, err := output.Create(unitName)
	if err != nil {
		return fmt.Errorf("create unit file %s: %w", unitName, err)
	}
	defer outputFile.Close()

	err = tpl.ExecuteTemplate(outputFile, unitName, nil)
	if err != nil {
		return fmt.Errorf("execute template for unit %s: %w", unitName, err)
	}

	return nil
}

// createLessonTemplate creates a template instance for a specific lesson with access to course-level templates
func createLessonTemplate(courseFS, lessonFS fs.FS) (*template.Template, error) {
	// Start with lesson content template (includes lesson templates and functions)
	tpl, err := createContentTemplate(lessonFS)
	if err != nil {
		return nil, fmt.Errorf("create lesson content template: %w", err)
	}

	// Parse course-level templates on top (excluding *.md to avoid content files)
	coursePatterns := []string{
		"templates/*.md",
	}

	tpl, err = parseTemplatePatterns(tpl, courseFS, coursePatterns)
	if err != nil {
		return nil, fmt.Errorf("parse course templates: %w", err)
	}

	return tpl, nil
}

// createUnitTemplate creates a template instance for a specific unit with access to training-level templates
func createUnitTemplate(trainingFS, unitsFS fs.FS) (*template.Template, error) {
	// Start with units content template (includes unit templates and functions)
	tpl, err := createContentTemplate(unitsFS)
	if err != nil {
		return nil, fmt.Errorf("create unit content template: %w", err)
	}

	// Parse training-level templates on top (excluding *.md to avoid content files)
	trainingPatterns := []string{
		"templates/*.md",
	}

	tpl, err = parseTemplatePatterns(tpl, trainingFS, trainingPatterns)
	if err != nil {
		return nil, fmt.Errorf("parse training templates: %w", err)
	}

	return tpl, nil
}
