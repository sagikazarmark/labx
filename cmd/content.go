package cmd

import (
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/goccy/go-yaml"
	"github.com/sagikazarmark/labx/labx"
	"github.com/sagikazarmark/labx/pkg/sproutx"
	"github.com/spf13/cobra"
)

type contentOptions struct {
	path    string
	channel string
}

func NewContentCommand() *cobra.Command {
	var opts contentOptions

	cmd := &cobra.Command{
		Use:   "content",
		Short: "Generate content",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContent(&opts, cmd.OutOrStdout())
		},
	}

	flags := cmd.Flags()

	flags.StringVar(
		&opts.path,
		"path",
		".",
		`Path to load manifest from`,
	)

	flags.StringVar(
		&opts.channel,
		"channel",
		"dev",
		`Which channel to push the playground to`,
	)

	return cmd
}

const betaNotice = `::remark-box
---
kind: warning
---

⚠️ This content is marked as **beta**, meaning it’s unfinished or still in progress and may change significantly.
::

`

func runContent(opts *contentOptions, output io.Writer) error {
	root, err := os.OpenRoot(opts.path)
	if err != nil {
		return err
	}

	fsys := root.FS()

	manifest, err := labx.Content(fsys, opts.channel)
	if err != nil {
		return err
	}

	// markdownFile, err := fsys.Open("index.md")
	// if err != nil {
	// 	return err
	// }
	// defer markdownFile.Close()

	encoder := yaml.NewEncoder(
		output,
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.IndentSequence(true),
	)

	_, err = io.WriteString(output, "---\n")
	if err != nil {
		return err
	}

	err = encoder.Encode(manifest)
	if err != nil {
		return err
	}

	_, err = io.WriteString(output, "---\n")
	if err != nil {
		return err
	}

	if strings.ToLower(opts.channel) == "beta" {
		_, err = io.WriteString(output, betaNotice)
		if err != nil {
			return err
		}
	}

	handler := sprout.New(sprout.WithRegistries(sproutx.NewRegistry()))
	funcs := handler.Build()

	tpl, err := template.New("index.md").Funcs(funcs).ParseFS(fsys, "index.md")
	if err != nil {
		return err
	}

	// tpl = tpl.Funcs(funcs)

	err = tpl.Execute(output, templateData{fsys})
	if err != nil {
		return err
	}

	// _, err = io.Copy(output, markdownFile)
	// if err != nil {
	// 	return err
	// }

	return nil
}

type templateData struct {
	Fsys fs.FS
}
