package labx

import (
	"os"

	"github.com/goccy/go-yaml"
)

// manifestKind represents a minimal manifest structure to determine routing
type manifestKind struct {
	Kind string `yaml:"kind" json:"kind"`
}

// Generate processes content based on the manifest kind, routing to appropriate handlers
func Generate(root *os.Root, output *os.Root, channel string) error {
	// Read and parse just the kind field from manifest.yaml
	manifestFile, err := root.FS().Open("manifest.yaml")
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	decoder := yaml.NewDecoder(manifestFile)

	var kind manifestKind
	err = decoder.Decode(&kind)
	if err != nil {
		return err
	}

	// Route based on kind
	if kind.Kind == "playground" {
		return generatePlayground(root, output, channel)
	}

	// Everything else goes through content processing
	return Content(root, output, channel)
}

// generatePlayground handles playground-specific generation
func generatePlayground(root *os.Root, output *os.Root, channel string) error {
	return Playground(root, output, channel)
}
