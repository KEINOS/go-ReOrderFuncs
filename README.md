# go-ReOrderFuncs

Alphabetically sorts test functions in Go source files.

<!-- WIP

A Go utility tool that automatically reorganizes test functions in Go source files alphabetically while preserving all other code structure, comments, and formatting.

## Installation

```bash
go install github.com/KEINOS/go-ReOrderFuncs/cmd/reorderfuncs@latest
```

## Usage

### Command Line Tool

```bash
# Reorder test functions in place
reorderfuncs myfile_test.go

# Reorder and save to a different file
reorderfuncs input_test.go output_test.go
```

### Library Usage

```go
package main

import (
    "log"
    reorderfuncs "github.com/KEINOS/go-ReOrderFuncs"
)

func main() {
    err := reorderfuncs.Exec("input_test.go", "output_test.go")
    if err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### Core Functions

#### `Exec(pathInput, pathOutput string) error`

Reorders test functions in a Go source file alphabetically.

- `pathInput`: Path to the input Go source file
- `pathOutput`: Path to write the reordered output (can be the same as input)
- Returns: Error if the operation fails

#### `ExtractTestFunctions(lines []string, file *ast.File, fset *token.FileSet) ([]TestFunction, []string)`

Extracts test functions from source lines using AST information.

- `lines`: Source file content split into lines
- `file`: Parsed AST file structure
- `fset`: Token file set for position information
- Returns: Slice of TestFunction structs and non-test content lines

### Types

#### `TestFunction`

Represents a test function with its content.

```go
type TestFunction struct {
    Name  string   // Name of the test function
    Lines []string // Source lines including comments
}
```

## Example

### Before

```go
package main

import "testing"

func Test_david(t *testing.T) {
    // Test implementation
    const whoami = "david"
    if whoami != "david" {
        t.Errorf("whoami is not david")
    }
}

func Test_bob(t *testing.T) {
    // Test implementation
    const whoami = "bob"
    if whoami != "bob" {
        t.Errorf("whoami is not bob")
    }
}

func Test_alice(t *testing.T) {
    // Test implementation
    const whoami = "alice"
    if whoami != "alice" {
        t.Errorf("whoami is not alice")
    }
}
```

### After

```go
package main

import "testing"

func Test_alice(t *testing.T) {
    // Test implementation
    const whoami = "alice"
    if whoami != "alice" {
        t.Errorf("whoami is not alice")
    }
}

func Test_bob(t *testing.T) {
    // Test implementation
    const whoami = "bob"
    if whoami != "bob" {
        t.Errorf("whoami is not bob")
    }
}

func Test_david(t *testing.T) {
    // Test implementation
    const whoami = "david"
    if whoami != "david" {
        t.Errorf("whoami is not david")
    }
}
```

## Development

### Requirements

- Go 1.21 or later
- golangci-lint (for development)

### Running Tests

```bash
go test ./...
```

### Test Coverage

```bash
go test -cover ./...
```

### Linting

```bash
golangci-lint run
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Follow TDD methodology (write tests first)
4. Ensure all tests pass and linting is clean
5. Submit a pull request

## License

This project is licensed under the MIT License.

-->
