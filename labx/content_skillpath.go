package labx

import (
	"fmt"
	"io/fs"
	"text/template"
)

// Skill paths accept upstream's sibling unit-*.md files as well as labx's units/ layout.
func renderSkillPathUnits(ctx renderContext, tpl *template.Template) error {
	units, err := fs.Glob(ctx.Root.FS(), "unit-*.md")
	if err != nil {
		return err
	}
	for _, unit := range units {
		if exists, err := fileExists(ctx.Root.FS(), "units/"+unit); err != nil {
			return err
		} else if exists {
			return fmt.Errorf("duplicate skill-path unit %s", unit)
		}
		if err := renderRootTemplate(ctx, tpl, unit); err != nil {
			return err
		}
	}
	return renderUnits(ctx)
}
