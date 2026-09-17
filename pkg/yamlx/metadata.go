// Package yamlx provides YAML decoding support for extensible manifest models.
package yamlx

import (
	"reflect"
	"strings"
)

// DecodeWithMetadata decodes a plain struct alias and retains only unmodeled
// fields in its inline map. go-yaml otherwise copies all keys into inline maps,
// including build-only fields that must not survive conversion.
// T must be a struct alias without UnmarshalYAML methods to avoid recursion.
func DecodeWithMetadata[T any](
	unmarshal func(any) error,
	target *T,
	metadata *map[string]any,
) error {
	if err := unmarshal(target); err != nil {
		return err
	}
	typ := reflect.TypeOf(*target)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if name != "" && name != "-" {
			delete(*metadata, name)
		}
	}
	if len(*metadata) == 0 {
		*metadata = nil
	}
	return nil
}
