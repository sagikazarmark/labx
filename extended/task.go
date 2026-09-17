package extended

import "strings"

// An omitted dimension leaves the corresponding field unset on one task,
// allowing Labs to choose its default machine or user.
func taskTargets(values StringList) []string {
	if values == nil {
		return []string{""}
	}
	return values
}

func taskName(base string, segments ...string) string {
	for _, segment := range segments {
		base += "_" + segment
	}

	return sanitizeTaskName(base)
}

func sanitizeTaskName(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}
