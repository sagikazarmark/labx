package labx

import (
	"fmt"
	"io/fs"
	"strings"
	"text/template"
)

// renderTraining handles training-specific rendering
func renderTraining(ctx renderContext, tpl *template.Template) error {
	fsys := ctx.Root.FS()

	// Process program.md if it exists
	hasProgramFile, err := fileExists(fsys, "program.md")
	if err != nil {
		return err
	}

	if hasProgramFile {
		return renderRootTemplate(ctx, tpl, "program.md")
	}

	// Copy static files if they exist at the training level
	hasStatic, err := dirExists(ctx.Root.FS(), "static")
	if err != nil {
		return err
	}

	if hasStatic {
		err = copyStaticFiles(ctx.Root, ctx.Output, "static", "__static__")
		if err != nil {
			return fmt.Errorf("copy static files: %w", err)
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

			err = renderTrainingUnit(ctx, "units", unitName)
			if err != nil {
				return fmt.Errorf("render unit %s: %w", unitName, err)
			}
		}
	}

	return nil
}

// renderTrainingUnit processes a unit file and renders its content
func renderTrainingUnit(ctx renderContext, unitPath, unitName string) error {
	fsys := ctx.Root.FS()

	// Create a sub-filesystem constrained to the units directory
	unitsFS, err := fs.Sub(fsys, unitPath)
	if err != nil {
		return fmt.Errorf("create units sub-filesystem: %w", err)
	}

	// Create unit-specific template instance with access to training-level templates
	tpl, err := createTrainingUnitTemplate(ctx.Root.FS(), unitsFS)
	if err != nil {
		return fmt.Errorf("create unit template: %w", err)
	}

	// Copy and process markdown files from units/ to the root
	outputFile, err := ctx.Output.Create(unitName)
	if err != nil {
		return fmt.Errorf("create unit file %s: %w", unitName, err)
	}
	defer outputFile.Close()

	data := templateData{
		Channel:  ctx.Channel,
		Manifest: ctx.Manifest,
		Extra:    ctx.Extra,
	}
	err = tpl.ExecuteTemplate(outputFile, unitName, data)
	if err != nil {
		return fmt.Errorf("execute template for unit %s: %w", unitName, err)
	}

	return nil
}

// createTrainingUnitTemplate creates a template instance for a specific unit with access to training-level templates
func createTrainingUnitTemplate(trainingFS, unitsFS fs.FS) (*template.Template, error) {
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
