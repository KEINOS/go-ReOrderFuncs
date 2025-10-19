/*
Example usage and golden tests for public functions in reorderfuncs package.

Edge cases and error handling tests are in reorderfuncs_test.go.
*/
package reorderfuncs_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	reorderfuncs "github.com/KEINOS/go-ReOrderFuncs"
)

//nolint:gosec // file inclusion is controlled by caller
func ExampleExec() {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "reorderfuncs_test_*")
	if err != nil {
		panic(err)
	}

	defer func() { _ = os.RemoveAll(tempDir) }()

	pathInput := filepath.Join("testdata", "test_sample1_before")
	pathOutput := filepath.Join(tempDir, "output_test.go") // Output in temp dir
	pathExpect := filepath.Join("testdata", "test_sample1_expect")

	// Re-order functions in the input file and save to output file
	err = reorderfuncs.Exec(pathInput, pathOutput)
	if err != nil {
		panic(err)
	}

	// Read and compare output
	expect, err := os.ReadFile(pathExpect)
	if err != nil {
		panic(err)
	}

	actual, err := os.ReadFile(pathOutput)
	if err != nil {
		panic(err)
	}

	if string(actual) == string(expect) {
		fmt.Println("OK")
	}

	// Output: OK
}

func ExampleParseGoFile() {
	// Temporary file for demonstration
	source := `package main

import "testing"

func Test_example(t *testing.T) {
	// Test implementation
}`

	tempFile, err := os.CreateTemp("", "example_*.go")
	if err != nil {
		panic(err)
	}

	defer func() { _ = os.Remove(tempFile.Name()) }()

	_, err = tempFile.WriteString(source)
	if err != nil {
		panic(err)
	}

	err = tempFile.Close()
	if err != nil {
		panic(err)
	}

	// Parse the Go file
	lines, file, fset, err := reorderfuncs.ParseGoFile(tempFile.Name())
	if err != nil {
		panic(err)
	}

	fmt.Printf("Lines count: %d\n", len(lines))
	fmt.Printf("Package name: %s\n", file.Name.Name)
	fmt.Printf("FileSet valid: %t\n", fset != nil)

	// Output:
	// Lines count: 7
	// Package name: main
	// FileSet valid: true
}

func ExampleBuildOutputContent() {
	// Sample test functions
	testFuncs := []reorderfuncs.TestFunction{
		{
			Name:  "Test_beta",
			Lines: []string{"func Test_beta() {", "	// beta test", "}"},
		},
		{
			Name:  "Test_alpha",
			Lines: []string{"func Test_alpha() {", "	// alpha test", "}"},
		},
	}

	nonTestLines := []string{
		"package main",
		"",
		"import \"testing\"",
	}

	// Build output content
	outputContent := reorderfuncs.BuildOutputContent(testFuncs, nonTestLines)

	lines := strings.Split(outputContent, "\n")
	fmt.Printf("Total lines: %d\n", len(lines))
	fmt.Printf("First line: %s\n", lines[0])
	fmt.Printf("Contains Test_alpha: %t\n", strings.Contains(outputContent, "Test_alpha"))
	fmt.Printf("Contains Test_beta: %t\n", strings.Contains(outputContent, "Test_beta"))

	// Output:
	// Total lines: 12
	// First line: package main
	// Contains Test_alpha: true
	// Contains Test_beta: true
}

func ExampleExtractTestFunctions() {
	// Sample Go source code with test functions
	source := `package main

import "testing"

func Test_charlie(t *testing.T) {
	// Test charlie
}

func Test_alice(t *testing.T) {
	// Test alice
}

func regularFunction() {
	// Not a test
}

func Test_bob(t *testing.T) {
	// Test bob
}`

	lines := strings.Split(source, "\n")

	// Parse AST to get function positions
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	testFuncs, nonTestLines := reorderfuncs.ExtractTestFunctions(lines, file, fset)

	fmt.Printf("Found %d test functions\n", len(testFuncs))

	for _, tf := range testFuncs {
		fmt.Printf("- %s\n", tf.Name)
	}

	fmt.Printf("Non-test lines: %d\n", len(nonTestLines))

	// Output:
	// Found 3 test functions
	// - Test_charlie
	// - Test_alice
	// - Test_bob
	// Non-test lines: 7
}
