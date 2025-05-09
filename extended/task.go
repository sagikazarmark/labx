package extended

import "strings"

func taskName(base string, segments ...string) string {
	for _, segment := range segments {
		base += "_" + segment
	}

	return sanitizeTaskName(base)
}

func sanitizeTaskName(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}
