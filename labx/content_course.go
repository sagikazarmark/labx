package labx

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/sagikazarmark/labx/core"
)

// lessonTemplateData holds the data passed to lesson template executions
type lessonTemplateData struct {
	Channel  string
	Manifest core.ContentManifest
	Name     string
	Course   core.ContentManifest
	Module   *core.ContentManifest
	Extra    map[string]any
}

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

		err = renderLesson(ctx, lessonPath, lessonName, nil)
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

		moduleManifest, err := loadYAMLFile[core.ContentManifest](
			fsys,
			modulePath+"/manifest.yaml",
		)
		if err != nil {
			return fmt.Errorf("decode module manifest: %w", err)
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

			err = renderLesson(ctx, lessonPath, outputPath, &moduleManifest)
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

	outputFile, err := createOutputFile(output, outputPath)
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
func renderLesson(
	ctx renderContext,
	lessonPath, outputPath string,
	moduleManifest *core.ContentManifest,
) error {
	fsys := ctx.Root.FS()

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

	return renderCourseDirectory(ctx, lessonPath, outputPath, lessonManifest, moduleManifest, false)
}

func renderCourseMarkdown(ctx renderContext, tpl *template.Template) error {
	err := renderCourseRootMarkdown(ctx, tpl)
	if err != nil {
		return err
	}

	fsys := ctx.Root.FS()

	hasLessons, err := dirExists(fsys, "lessons")
	if err != nil {
		return err
	}

	hasModules, err := dirExists(fsys, "modules")
	if err != nil {
		return err
	}

	if hasLessons && hasModules {
		return fmt.Errorf("course cannot have both 'lessons' and 'modules' directories")
	}

	if hasLessons {
		return renderCourseLessonsMarkdown(ctx)
	}

	if hasModules {
		return renderCourseModulesMarkdown(ctx)
	}

	return nil
}

func renderCourseRootMarkdown(ctx renderContext, tpl *template.Template) error {
	rootFiles, err := fs.ReadDir(ctx.Root.FS(), ".")
	if err != nil {
		return err
	}

	for _, file := range rootFiles {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".md") {
			continue
		}

		err = renderTemplate(
			rootedOutput{ctx.Output},
			fileName,
			tpl,
			fileName,
			newTemplateData(ctx),
		)
		if err != nil {
			return fmt.Errorf("execute template %s: %w", fileName, err)
		}
	}

	return nil
}

func renderCourseLessonsMarkdown(ctx renderContext) error {
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

		lessonFS, err := fs.Sub(fsys, lessonPath)
		if err != nil {
			return fmt.Errorf("create lesson sub-filesystem: %w", err)
		}

		lessonManifest, err := loadContentManifestOrDefault(lessonFS, ctx.Manifest)
		if err != nil {
			return fmt.Errorf("load lesson manifest %s: %w", lessonName, err)
		}

		err = renderCourseDirectory(ctx, lessonPath, lessonName, lessonManifest, nil, true)
		if err != nil {
			return fmt.Errorf("render lesson %s: %w", lessonName, err)
		}
	}

	return nil
}

func renderCourseModulesMarkdown(ctx renderContext) error {
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

		moduleFS, err := fs.Sub(fsys, modulePath)
		if err != nil {
			return fmt.Errorf("create module sub-filesystem: %w", err)
		}

		moduleManifest, found, err := loadContentManifestIfExists(moduleFS)
		if err != nil {
			return fmt.Errorf("load module manifest %s: %w", moduleName, err)
		}

		currentManifest := ctx.Manifest
		var parentModule *core.ContentManifest
		if found {
			currentManifest = moduleManifest
			parentModule = &moduleManifest
		}

		err = renderCourseDirectory(ctx, modulePath, moduleName, currentManifest, nil, true)
		if err != nil {
			return fmt.Errorf("render module %s: %w", moduleName, err)
		}

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

			lessonFS, err := fs.Sub(fsys, lessonPath)
			if err != nil {
				return fmt.Errorf("create lesson sub-filesystem: %w", err)
			}

			lessonManifest, err := loadContentManifestOrDefault(lessonFS, currentManifest)
			if err != nil {
				return fmt.Errorf("load lesson manifest %s: %w", lessonName, err)
			}

			err = renderCourseDirectory(
				ctx,
				lessonPath,
				outputPath,
				lessonManifest,
				parentModule,
				true,
			)
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

func renderCourseDirectory(
	ctx renderContext,
	directoryPath, outputPath string,
	manifest core.ContentManifest,
	moduleManifest *core.ContentManifest,
	includeIndex bool,
) error {
	err := ensureOutputDir(ctx.Output, outputPath)
	if err != nil {
		return err
	}

	lessonFS, err := fs.Sub(ctx.Root.FS(), directoryPath)
	if err != nil {
		return fmt.Errorf("create lesson sub-filesystem: %w", err)
	}

	tpl, err := createLessonTemplate(ctx.Root.FS(), lessonFS, ctx.BaseTemplate)
	if err != nil {
		return fmt.Errorf("create directory template: %w", err)
	}

	lessonFiles, err := fs.ReadDir(lessonFS, ".")
	if err != nil {
		return err
	}

	for _, file := range lessonFiles {
		if file.IsDir() {
			if file.Name() == "static" {
				err = copyStaticFiles(
					ctx.Root,
					ctx.Output,
					directoryPath+"/static",
					outputPath+"/__static__",
				)
				if err != nil {
					return fmt.Errorf("copy static files: %w", err)
				}
			}
			continue
		}

		fileName := file.Name()
		if strings.HasSuffix(fileName, ".md") && (includeIndex || fileName != "index.md") {
			outputFilePath := outputPath + "/" + fileName

			data := lessonTemplateData{
				Channel:  ctx.Channel,
				Manifest: manifest,
				Name:     ctx.Name,
				Course:   ctx.Manifest,
				Module:   moduleManifest,
				Extra:    ctx.Extra,
			}

			err = renderTemplate(rootedOutput{ctx.Output}, outputFilePath, tpl, fileName, data)
			if err != nil {
				return fmt.Errorf("execute template %s: %w", fileName, err)
			}
		}
	}

	return nil
}

func loadContentManifestIfExists(fsys fs.FS) (core.ContentManifest, bool, error) {
	hasManifest, err := fileExists(fsys, "manifest.yaml")
	if err != nil {
		return core.ContentManifest{}, false, err
	}

	if !hasManifest {
		return core.ContentManifest{}, false, nil
	}

	manifest, err := loadYAMLFile[core.ContentManifest](fsys, "manifest.yaml")

	return manifest, true, err
}

func loadContentManifestOrDefault(
	fsys fs.FS,
	fallback core.ContentManifest,
) (core.ContentManifest, error) {
	manifest, found, err := loadContentManifestIfExists(fsys)
	if err != nil {
		return core.ContentManifest{}, err
	}

	if found {
		return manifest, nil
	}

	return fallback, nil
}

// createLessonTemplate creates a template instance for a specific lesson with access to course-level templates
func createLessonTemplate(
	courseFS, lessonFS fs.FS,
	baseTemplate *template.Template,
) (*template.Template, error) {
	// Clone the global template to avoid conflicts
	tpl, err := baseTemplate.Clone()
	if err != nil {
		return nil, fmt.Errorf("clone global template: %w", err)
	}

	tpl = tpl.Funcs(createTemplateFuncs(lessonFS))

	// Parse course-level templates on top (excluding *.md to avoid content files)
	coursePatterns := []string{
		"templates/*.md",
	}

	tpl, err = parseTemplatePatterns(tpl, courseFS, coursePatterns)
	if err != nil {
		return nil, fmt.Errorf("parse course templates: %w", err)
	}

	// Parse lesson content patterns
	lessonPatterns := []string{
		"*.md",
		"templates/*.md",
	}

	tpl, err = parseTemplatePatterns(tpl, lessonFS, lessonPatterns)
	if err != nil {
		return nil, fmt.Errorf("create lesson content template: %w", err)
	}

	return tpl, nil
}
