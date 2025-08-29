package labx

import (
	"fmt"
	"io/fs"
	"os"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"

	"github.com/sagikazarmark/labx/pkg/sproutx"
)

func renderRootTemplate(ctx renderContext, tpl *template.Template, name string) error {
	data := templateData{
		Channel:  ctx.Channel,
		Manifest: ctx.Manifest,
		Extra:    ctx.Extra,
	}

	return renderTemplate(ctx.Output, name, tpl, name, data)
}

func renderTemplate(
	output *os.Root,
	outputPath string,
	tpl *template.Template,
	name string,
	data any,
) error {
	outputFile, err := output.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	return tpl.ExecuteTemplate(outputFile, name, data)
}

func createBaseTemplate(rootFS fs.FS, templateFSs []fs.FS) (*template.Template, error) {
	tplFuncs := createTemplateFuncs(rootFS)
	tpl := template.New("").Funcs(tplFuncs)

	for _, templateFS := range templateFSs {
		patterns := []string{
			"*.md",
			"**/*.md",
		}

		var err error
		tpl, err = parseTemplatePatterns(tpl, templateFS, patterns)
		if err != nil {
			return nil, fmt.Errorf("parse templates: %w", err)
		}
	}

	return tpl, nil
}

// createTemplateFuncs creates template functions for the given filesystem
func createTemplateFuncs(fsys fs.FS) template.FuncMap {
	return sprout.New(
		sprout.WithRegistries(
			sproutx.NewFSRegistry(fsys),
			sproutx.NewStringsRegistry(),
		),
		sprout.WithGroups(all.RegistryGroup()),
	).Build()
}

// parseTemplatePatterns parses template patterns from a filesystem into a template
func parseTemplatePatterns(
	tpl *template.Template,
	fsys fs.FS,
	patterns []string,
) (*template.Template, error) {
	var matchedPatterns []string

	for _, pattern := range patterns {
		matches, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, err
		}

		if len(matches) == 0 {
			continue
		}

		matchedPatterns = append(matchedPatterns, pattern)
	}

	if len(matchedPatterns) == 0 {
		return tpl, nil
	}

	return tpl.ParseFS(fsys, matchedPatterns...)
}
