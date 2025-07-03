# Sprout FS Registry

This package provides a Sprout registry for file system operations, extending the [go-sprout](https://github.com/go-sprout/sprout) template engine with file reading capabilities.

## Functions

### readFile

Reads the entire content of a file.

```go
{{ readFile "path/to/file.txt" }}
```

### readFileRange

Reads a specific range of lines from a file.

```go
{{ readFileRange "path/to/file.txt" 5 10 }}
```

### readFileUntil

Reads lines from the beginning of a file until a specified line number.

```go
{{ readFileUntil "path/to/file.txt" 10 }}
```

### readFileLine

Reads a specific line from a file.

```go
{{ readFileLine "path/to/file.txt" 5 }}
```

### readFileBlock

Reads a named block from a file. This is useful for extracting specific sections of code or configuration that are marked with block delimiters.

```go
{{ readFileBlock "path/to/file.txt" "block-name" }}
```

## Block Syntax

The `readFileBlock` function uses a language-independent block syntax that works within comments of any programming language:

### Block Start
```
@block:block-name
```

### Block End
```
@endblock
```
or optionally with the block name:
```
@endblock:block-name
```

The block markers should be embedded within comments appropriate for the file's language. The block name in `@endblock` is optional but can be useful for clarity in complex files.

## Examples

### Go Code
```go
// @block:imports
import (
    "fmt"
    "os"
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
}
```

### Bash Script
```bash
# @block:variables
export API_URL="https://api.example.com"
export MAX_RETRIES=3
export TIMEOUT=30
# @endblock:variables

# @block:functions
function log_info() {
    echo "[INFO] $(date): $1"
}

function retry_command() {
    local command="$1"
    local max_attempts="$2"
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        log_info "Attempt $attempt of $max_attempts: $command"
        if eval "$command"; then
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    log_error "Command failed after $max_attempts attempts: $command"
    return 1
}
# @endblock
```

### HTML
```html
<!-- @block:header -->
<head>
    <title>Example Page</title>
    <meta charset="UTF-8">
</head>
<!-- @endblock:header -->
```

### SQL
```sql
-- @block:user_query
SELECT name, email
FROM users
WHERE active = 1
ORDER BY created_at DESC;
-- @endblock:user_query
```

## Usage

```go
package main

import (
    "embed"
    "text/template"
    
    "github.com/go-sprout/sprout"
    "github.com/sagikazarmark/labx/pkg/sproutx"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed data/*
var dataFS embed.FS

func main() {
    // Create the FS registry
    fsRegistry := sproutx.NewFSRegistry(dataFS)
    
    // Create a new sprout function map
    funcMap := sprout.FuncMap()
    
    // Register the FS functions
    fsRegistry.RegisterFunctions(funcMap)
    
    // Use with text/template
    tmpl := template.Must(template.New("example").Funcs(funcMap).ParseFS(templateFS, "templates/*"))
    
    // Now you can use readFileBlock in your templates
    // {{ readFileBlock "testdata/fs/config.yaml" "database" }}
}
```

## Features

- **Language Independent**: Works with any comment style (`//`, `#`, `/*`, `<!--`, `--`, `%`, `;`, etc.)
- **Flexible End Markers**: Supports generic (`@endblock`) and optionally named (`@endblock:name`) end markers
- **Error Handling**: Provides clear error messages for missing blocks or unclosed blocks
- **Whitespace Handling**: Preserves original formatting and indentation
- **Multiple Blocks**: If multiple blocks with the same name exist, returns the first one found
- **Bash Script Support**: Works seamlessly with bash scripts and shell comments

## Error Conditions

The function will return an error in the following cases:
- File not found
- Block not found in the file
- Block is not properly closed (missing `@endblock`)
- File reading errors

## Design Goals

The block syntax was designed to be:
- **Comment-friendly**: Works within any comment style
- **Template-safe**: Doesn't conflict with Go template syntax (`{{ }}`)
- **Readable**: Clear and intuitive syntax
- **Flexible**: Block end markers don't require naming but support it optionally
- **Robust**: Handles edge cases like whitespace and special characters in block names
- **Universal**: Works across all programming languages and configuration files