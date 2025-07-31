package labx

import "text/template"

// renderChallenge handles challenge-specific rendering
func renderChallenge(ctx renderContext, tpl *template.Template) error {
	hasSolution, err := fileExists(ctx.Root.FS(), "solution.md")
	if err != nil {
		return err
	}

	if hasSolution {
		return renderRootTemplate(ctx, tpl, "solution.md")
	}

	return nil
}
