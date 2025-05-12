package iox

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ReadLinesRange reads lines from a [io.Reader] in the specified range (or until EOF).
func ReadLinesRange(r io.Reader, from, to int) (string, error) {
	if from <= 0 || to < from {
		return "", fmt.Errorf("invalid line range: from=%d to=%d", from, to)
	}

	var builder strings.Builder

	reader := bufio.NewReader(r)
	lineNumber := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("error reading: %w", err)
		}

		// Remove trailing newline characters
		line = strings.TrimRight(line, "\r\n")
		lineNumber++

		if lineNumber >= from && lineNumber <= to {
			builder.WriteString(line)
		}

		if err == io.EOF || lineNumber >= to {
			break
		}

		if lineNumber >= from && lineNumber <= to {
			builder.WriteByte('\n') // re-add newline for correct formatting
		}
	}

	return builder.String(), nil
}
