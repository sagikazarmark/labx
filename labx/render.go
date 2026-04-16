package labx

import (
	"fmt"

	"github.com/iximiuz/labctl/api"
	"github.com/iximiuz/labctl/content"

	"github.com/sagikazarmark/labx/core"
)

func Render(opts GenerateOpts) error {
	kind, ctx, err := prepareGenerateContext(opts)
	if err != nil {
		return err
	}

	if kind.Kind == "playground" {
		return renderPlaygroundContent(ctx)
	}

	return renderContentOnly(ctx)
}

func renderPlaygroundContent(ctx GenerateContext) error {
	manifest, err := loadYAMLFile[api.PlaygroundManifest](ctx.Root.FS(), "manifest.yaml")
	if err != nil {
		return err
	}

	markdown, found, err := renderPlaygroundMarkdown(
		ctx.Root.FS(),
		ctx.Channel,
		manifest,
		ctx.BaseTemplate,
		ctx.ExtraData,
	)
	if err != nil {
		return err
	}

	if found {
		err = writeStringFile(ctx.Output, "README.md", markdown)
		if err != nil {
			return err
		}
	}

	return copyStaticFilesIfExists(ctx.Root, ctx.Output, "static", "__static__")
}

func renderContentOnly(ctx GenerateContext) error {
	manifest, err := loadYAMLFile[core.ContentManifest](ctx.Root.FS(), "manifest.yaml")
	if err != nil {
		return err
	}

	tpl, err := createContentTemplateFromGlobal(ctx.BaseTemplate, ctx.Root.FS())
	if err != nil {
		return err
	}

	renderCtx := newRenderContext(ctx, manifest, manifest.Name)

	switch manifest.Kind {
	case content.KindChallenge:
		err = renderRootTemplate(renderCtx, tpl, "index.md")
		if err != nil {
			return err
		}

		err = renderSolution(renderCtx, tpl)
		if err != nil {
			return err
		}
	case content.KindTutorial:
		err = renderRootTemplate(renderCtx, tpl, "index.md")
		if err != nil {
			return err
		}
	case content.KindTraining:
		_, err = renderProgram(renderCtx, tpl)
		if err != nil {
			return err
		}

		err = renderUnits(renderCtx)
		if err != nil {
			return err
		}
	case content.KindCourse:
		err = renderCourseMarkdown(renderCtx, tpl)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported content kind %q", manifest.Kind)
	}

	return copyStaticFilesIfExists(ctx.Root, ctx.Output, "static", "__static__")
}
