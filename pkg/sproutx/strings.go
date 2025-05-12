package sproutx

import (
	"strings"

	"github.com/go-sprout/sprout"
)

// StringsRegistry struct implements the [sprout.Registry] interface, embedding the Handler to access shared functionalities.
type StringsRegistry struct {
	handler sprout.Handler
}

// NewStringsRegistry initializes and returns a new [sprout.Registry].
func NewStringsRegistry() *StringsRegistry {
	return &StringsRegistry{}
}

// Implements [sprout.Registry].
func (r *StringsRegistry) UID() string {
	return "sagikazarmark/labx.strings"
}

// Implements [sprout.Registry].
func (r *StringsRegistry) LinkHandler(fh sprout.Handler) error {
	r.handler = fh

	return nil
}

// Implements [sprout.Registry].
func (r *StringsRegistry) RegisterFunctions(funcsMap sprout.FunctionMap) error {
	sprout.AddFunction(funcsMap, "undindentSmart", r.UnindentSmart)

	return nil
}

func (r *StringsRegistry) UnindentSmart(value string) string {
	lines := strings.Split(value, "\n")
	minIndent := -1

	// Determine the minimum indentation of all non-empty lines
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" {
			continue // skip empty lines
		}
		indent := len(line) - len(trimmed)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	// If there's no indentation to remove, return the original string
	if minIndent <= 0 {
		return value
	}

	// Remove the minimum indentation from all lines
	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}

	return strings.Join(lines, "\n")
}
