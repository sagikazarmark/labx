package main

import "fmt"

// @block:imports
import (
	"os"
	"strings"
)
// @endblock:imports

func main() {
	// @block:setup
	name := "World"
	if len(os.Args) > 1 {
		name = strings.Join(os.Args[1:], " ")
	}
	// @endblock:setup

	fmt.Println("Hello, " + name + "!")

	// @block:cleanup
	// This is where cleanup code would go
	// Multiple lines are supported
	defer func() {
		fmt.Println("Goodbye!")
	}()
	// @endblock:cleanup
}

/*
@block:documentation
This is a multi-line block that demonstrates
how the block syntax works with different
comment styles.

The block markers work in:
- Single-line comments (//)
- Multi-line comments (/* */)
- Shell-style comments (#)
- Any other comment format
@endblock:documentation
*/

// @block:constants
const (
	MaxRetries = 3
	DefaultTimeout = 30
)
// @endblock
