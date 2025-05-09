package extended

import (
	"fmt"

	"github.com/goccy/go-yaml/ast"
)

type StringList []string

func (s *StringList) UnmarshalYAML(node ast.Node) error {
	switch n := node.(type) {
	case *ast.StringNode:
		*s = []string{n.Value}

		return nil

	case *ast.SequenceNode:
		result := make([]string, 0, len(n.Values))

		for _, elem := range n.Values {
			strNode, ok := elem.(*ast.StringNode)
			if !ok {
				return fmt.Errorf("sequence element is not a string: %#v", elem)
			}

			result = append(result, strNode.Value)
		}

		*s = result

		return nil

	default:
		return fmt.Errorf("unsupported YAML node type: %T", node)
	}
}
