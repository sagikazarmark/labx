package labx

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"
)

// renderCourse handles rendering for both simple and modular courses
func renderCourse(ctx renderContext) error {
	fsys := ctx.Root.FS()

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
		return renderSimpleCourse(ctx)
	} else if hasModules {
		return renderModularCourse(ctx)
	}

	return nil
}

// renderSimpleCourse handles simple courses with a lessons directory
func renderSimpleCourse(ctx renderContext) error {
	fsys := ctx.Root.FS()

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

		err = renderLesson(ctx, lessonPath, lessonName)
		if err != nil {
			return fmt.Errorf("render lesson %s: %w", lessonName, err)
		}
	}

	return nil
}

// renderModularCourse handles modular courses with a modules directory
func renderModularCourse(ctx renderContext) error {
	fsys := ctx.Root.FS()

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
		err = renderModuleManifest(ctx.Root, ctx.Output, modulePath, moduleName)
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

			err = renderLesson(ctx, lessonPath, outputPath)
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
func renderLesson(ctx renderContext, lessonPath, outputPath string) error {
	fsys := ctx.Root.FS()

	// Create lesson directory first
	err := ctx.Output.Mkdir(outputPath, 0o755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Create a sub-filesystem constrained to the lesson directory
	lessonFS, err := fs.Sub(fsys, lessonPath)
	if err != nil {
		return fmt.Errorf("create lesson sub-filesystem: %w", err)
	}

	// Convert lesson manifest once and reuse
	lessonManifest, err := convertContentManifest(lessonFS, ctx.Channel)
	if err != nil {
		return fmt.Errorf("convert lesson manifest: %w", err)
	}

	// Process lesson manifest through the same pipeline as other manifests
	err = renderManifest(ctx.Output, outputPath+"/00-index.md", lessonManifest)
	if err != nil {
		return err
	}

	// Create lesson-specific template instance with access to course-level templates
	tpl, err := createLessonTemplate(ctx.Root.FS(), lessonFS)
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
				err = copyStaticFiles(
					ctx.Root,
					ctx.Output,
					lessonPath+"/static",
					outputPath+"/__static__",
				)
				if err != nil {
					return fmt.Errorf("copy static files: %w", err)
				}
			}
			continue
		}

		fileName := file.Name()
		if strings.HasSuffix(fileName, ".md") && fileName != "index.md" {
			outputFilePath := outputPath + "/" + fileName

			data := templateData{
				Channel:  ctx.Channel,
				Manifest: lessonManifest,
				Extra:    ctx.Extra,
			}

			err = renderTemplate(ctx.Output, outputFilePath, tpl, fileName, data)
			if err != nil {
				return fmt.Errorf("execute template %s: %w", fileName, err)
			}
		}
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
