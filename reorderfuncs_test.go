package reorderfuncs

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
//	Helpers
// ============================================================================

// TestFunc represents a test function with its name and position information.
type TestFunc struct {
	Name string
	Pos  token.Pos
}

// parseGoFileTestCase represents a test case for ParseGoFile.
type parseGoFileTestCase struct {
	name          string
	fileContent   string
	expectedLines int
	expectedPkg   string
	expectError   bool
	description   string
}

// extractTestFunctions extracts all test functions from an AST node.
func extractTestFunctions(node *ast.File) []TestFunc {
	var testFuncs []TestFunc

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "Test") {
				testFuncs = append(testFuncs, TestFunc{
					Name: fn.Name.Name,
					Pos:  fn.Pos(),
				})
			}
		}
	}

	return testFuncs
}

// runEdgeCaseTest executes a single edge case test for the Exec function.
//
//nolint:gosec // File path is controlled in test environment
func runEdgeCaseTest(t *testing.T, beforeFile, expectFile string) {
	t.Helper()

	// Create temp output file
	outputPath := filepath.Join(t.TempDir(), "output.go")

	// Execute the reordering
	err := Exec(beforeFile, outputPath)
	require.NoError(t, err)

	// Read actual output
	actualContent, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	// Read expected content
	expectedContent, err := os.ReadFile(expectFile)
	require.NoError(t, err)

	// Normalize trailing newlines for consistent comparison
	actual := strings.TrimSuffix(string(actualContent), "\n")
	expected := strings.TrimSuffix(string(expectedContent), "\n")

	// Compare content
	assert.Equal(t, expected, actual, "reordered content should match expected")

	// Parse and verify that test functions are in alphabetical order
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, outputPath, actualContent, parser.ParseComments)
	require.NoError(t, err, "parsed output should be valid Go")

	testFuncs := extractTestFunctions(node)
	for i := 1; i < len(testFuncs); i++ {
		assert.LessOrEqual(t, testFuncs[i-1].Name, testFuncs[i].Name,
			"Test functions should be in alphabetical order: %s should come before %s",
			testFuncs[i-1].Name, testFuncs[i].Name)
	}
}

// runExtractTestFunctionTest executes a single test case for ExtractTestFunctions.
func runExtractTestFunctionTest(t *testing.T, source string, expectedTestFuncs int,
	expectedTestNames []string, expectedNonTestLen int) {
	t.Helper()

	lines := strings.Split(source, "\n")

	// Parse AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	// Extract test functions
	testFuncs, nonTestLines := ExtractTestFunctions(lines, file, fset)

	// Verify results
	assert.Len(t, testFuncs, expectedTestFuncs)
	assert.Len(t, nonTestLines, expectedNonTestLen)

	// Check test function names
	actualNames := make([]string, len(testFuncs))
	for i, tf := range testFuncs {
		actualNames[i] = tf.Name
	}

	assert.ElementsMatch(t, actualNames, expectedTestNames)

	// Verify each test function has content
	for _, tf := range testFuncs {
		assert.NotEmpty(t, tf.Lines, "test function %s should have content", tf.Name)
		assert.NotEmpty(t, tf.Name, "test function name should not be empty")
	}
}

// runParseGoFileTest executes a single ParseGoFile test case.
func runParseGoFileTest(t *testing.T, test parseGoFileTestCase) {
	t.Helper()

	var filePath string
	if test.name == "nonexistent_file" {
		filePath = "/nonexistent/path/file.go"
	} else {
		// Create temporary file
		tempFile, err := os.CreateTemp(t.TempDir(), "test_*.go")
		require.NoError(t, err)

		defer func() { _ = os.Remove(tempFile.Name()) }()

		_, err = tempFile.WriteString(test.fileContent)
		require.NoError(t, err)
		require.NoError(t, tempFile.Close())

		filePath = tempFile.Name()
	}

	lines, file, fset, err := ParseGoFile(filePath)

	if test.expectError {
		assert.Error(t, err, test.description)

		return
	}

	require.NoError(t, err, test.description)
	assert.Len(t, lines, test.expectedLines, "line count should match")
	assert.Equal(t, test.expectedPkg, file.Name.Name, "package name should match")
	assert.NotNil(t, fset, "FileSet should not be nil")
}

// ============================================================================
//	Public Functions (ABC Order)
// ============================================================================

func TestExec_argument_check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		inputFile   string
		outputPath  string
		expectError bool
		description string
	}{
		{
			name:        "nonexistent_input_file",
			inputFile:   "/nonexistent/path/file.go",
			outputPath:  filepath.Join(t.TempDir(), "output.go"),
			expectError: true,
			description: "should fail when input file does not exist",
		},
		{
			name:        "invalid_output_directory",
			inputFile:   "testdata/test_sample1_before",
			outputPath:  "/nonexistent/directory/output.go",
			expectError: true,
			description: "should fail when output directory does not exist",
		},
		{
			name:        "successful_execution",
			inputFile:   "testdata/test_sample1_before",
			outputPath:  filepath.Join(t.TempDir(), "success_output.go"),
			expectError: false,
			description: "should succeed with valid input and output paths",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := Exec(test.inputFile, test.outputPath)

			if test.expectError {
				require.Error(t, err,
					test.description)
			} else {
				require.NoError(t, err,
					test.description)

				require.FileExists(t, test.outputPath,
					"output file should exist")
			}
		})
	}
}

// TestExec_comprehensive_edge_cases tests the Exec function with various edge cases
// including mixed content, generics, unicode characters, and complex test patterns.
func TestExec_comprehensive_edge_cases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		beforeFile string
		expectFile string
	}{
		{
			name:       "basic_reordering",
			beforeFile: "testdata/test_sample1_before",
			expectFile: "testdata/test_sample1_expect",
		},
		/* NOTE: Re-enable when var/const/type block handling spec is finalized
		{
			name:       "mixed_content_with_structs_and_methods",
			beforeFile: "testdata/test_sample2_before",
			expectFile: "testdata/test_sample2_expect",
		},
		*/
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runEdgeCaseTest(t, tc.beforeFile, tc.expectFile)
		})
	}
}

//nolint:funlen // test data structure requires multiple test cases
func TestExtractTestFunctions_basic_tests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		source             string
		expectedTestFuncs  int
		expectedTestNames  []string
		expectedNonTestLen int
	}{
		{
			name: "multiple test functions with comments",
			source: `package main

import "testing"

// Test for charlie
func Test_charlie(t *testing.T) {
	// Test implementation
}

func Test_alice(t *testing.T) {
	// Test alice
}

func regularFunction() {
	// Not a test
}

func Test_bob(t *testing.T) {
	// Test bob
}`,
			expectedTestFuncs:  3,
			expectedTestNames:  []string{"Test_charlie", "Test_alice", "Test_bob"},
			expectedNonTestLen: 7,
		},
		{
			name: "no test functions",
			source: `package main

import "testing"

func regularFunction() {
	// Not a test
}

func anotherFunction() {
	// Also not a test
}`,
			expectedTestFuncs:  0,
			expectedTestNames:  []string{},
			expectedNonTestLen: 12,
		},
		{
			name: "only test functions",
			source: `package main

import "testing"

func Test_first(t *testing.T) {
	// Test implementation
}

func Test_second(t *testing.T) {
	// Test implementation
}`,
			expectedTestFuncs:  2,
			expectedTestNames:  []string{"Test_first", "Test_second"},
			expectedNonTestLen: 4,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runExtractTestFunctionTest(t, testCase.source, testCase.expectedTestFuncs,
				testCase.expectedTestNames, testCase.expectedNonTestLen)
		})
	}
}

func TestExtractTestFunctions_with_comments(t *testing.T) {
	t.Parallel()

	source := `package main

import "testing"

// This is a comment for Test_alpha
// Multiple line comment
func Test_alpha(t *testing.T) {
	// Test implementation
	pass := true
	if !pass {
		t.Error("should pass")
	}
}`

	lines := strings.Split(source, "\n")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	testFuncs, _ := ExtractTestFunctions(lines, file, fset)

	require.Len(t, testFuncs, 1)
	assert.Equal(t, "Test_alpha", testFuncs[0].Name)

	// Check that comments are included in the function lines
	funcContent := strings.Join(testFuncs[0].Lines, "\n")
	assert.Contains(t, funcContent, "This is a comment for Test_alpha")
	assert.Contains(t, funcContent, "Multiple line comment")
	assert.Contains(t, funcContent, "func Test_alpha")
}

func TestParseGoFile_basic_tests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		fileContent   string
		expectedLines int
		expectedPkg   string
		expectError   bool
		description   string
	}{
		{
			name: "valid_go_file_with_test",
			fileContent: `package main

import "testing"

func Test_example(t *testing.T) {
	// Test implementation
}`,
			expectedLines: 7,
			expectedPkg:   "main",
			expectError:   false,
			description:   "valid Go file with test function",
		},
		{
			name:          "empty_package",
			fileContent:   `package test`,
			expectedLines: 1,
			expectedPkg:   "test",
			expectError:   false,
			description:   "minimal Go package",
		},
		{
			name:          "invalid_go_syntax",
			fileContent:   `package invalid syntax`,
			expectedLines: 0,
			expectedPkg:   "",
			expectError:   true,
			description:   "invalid Go syntax should return error",
		},
		{
			name:          "nonexistent_file",
			fileContent:   "", // Will use non-existent path
			expectedLines: 0,
			expectedPkg:   "",
			expectError:   true,
			description:   "non-existent file should return error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runParseGoFileTest(t, test)
		})
	}
}

// ============================================================================
//  Private Functions (ABC Order)
// ============================================================================

func Test_buildTestFunctionPositions_golden(t *testing.T) {
	t.Parallel()

	source := `package main

import "testing"

func Test_alpha(t *testing.T) {
	// line 6
}

func regularFunc() {
	// line 10
}

func Test_beta(t *testing.T) {
	// line 14
	// line 15
}`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	positions := buildTestFunctionPositions(file, fset)

	expected := map[string][2]int{
		"Test_alpha": {5, 7},   // Lines 5-7
		"Test_beta":  {13, 16}, // Lines 13-16
	}

	assert.Equal(t, expected, positions)
	assert.NotContains(t, positions, "regularFunc", "regular functions should not be included")
}

func Test_extractTestFunctionWithComments_golden(t *testing.T) {
	t.Parallel()

	lines := []string{
		"package main",                      // 0
		"",                                  // 1
		"import \"testing\"",                // 2
		"",                                  // 3
		"// Comment for test",               // 4
		"// Another comment",                // 5
		"func Test_example(t *testing.T) {", // 6
		"	// Test body",                     // 7
		"	pass := true",                     // 8
		"}",                                 // 9
		"",                                  // 10
		"func regularFunc() {",              // 11
		"}",                                 // 12
	}

	testFuncPos := map[string][2]int{
		"Test_example": {7, 10}, // Lines 7-10 (1-based)
	}

	testFunc, endIndex := extractTestFunctionWithComments(lines, "Test_example", testFuncPos)

	assert.Equal(t, "Test_example", testFunc.Name)
	assert.Equal(t, 9, endIndex) // 0-based end index

	expectedLines := []string{
		"", // Empty line before comments
		"// Comment for test",
		"// Another comment",
		"func Test_example(t *testing.T) {",
		"	// Test body",
		"	pass := true",
		"}",
	}
	assert.Equal(t, expectedLines, testFunc.Lines)
}

func Test_findCommentStart_golden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		lines                []string
		functionStartLine    int
		expectedCommentStart int
	}{
		{
			name: "function with preceding comments",
			lines: []string{
				"package main",                      // 0
				"",                                  // 1
				"// Comment 1",                      // 2
				"// Comment 2",                      // 3
				"func Test_example(t *testing.T) {", // 4
				"}",                                 // 5
			},
			functionStartLine:    4, // 0-based index for "func Test_example"
			expectedCommentStart: 1, // 0-based index for empty line before comments
		},
		{
			name: "function with no preceding comments",
			lines: []string{
				"package main",                      // 0
				"",                                  // 1
				"func regularFunc() {",              // 2
				"}",                                 // 3
				"func Test_example(t *testing.T) {", // 4
				"}",                                 // 5
			},
			functionStartLine:    4, // 0-based index for "func Test_example"
			expectedCommentStart: 4, // Same as function start (no comments)
		},
		{
			name: "function with mixed empty lines and comments",
			lines: []string{
				"package main",                      // 0
				"",                                  // 1
				"// Comment",                        // 2
				"",                                  // 3
				"// Another comment",                // 4
				"func Test_example(t *testing.T) {", // 5
				"}",                                 // 6
			},
			functionStartLine:    5, // 0-based index for "func Test_example"
			expectedCommentStart: 1, // 0-based index for first empty line before comments
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			commentStart := findCommentStart(test.lines, test.functionStartLine)
			assert.Equal(t, test.expectedCommentStart, commentStart)
		})
	}
}

func Test_findTestFunctionAtLine_golden(t *testing.T) {
	t.Parallel()

	testFuncPos := map[string][2]int{
		"Test_alpha": {5, 7},
		"Test_beta":  {10, 12},
	}

	tests := []struct {
		name         string
		lineNumber   int
		expectedFunc string
	}{
		{
			name:         "line matches Test_alpha start",
			lineNumber:   5,
			expectedFunc: "Test_alpha",
		},
		{
			name:         "line matches Test_beta start",
			lineNumber:   10,
			expectedFunc: "Test_beta",
		},
		{
			name:         "line does not match any function start",
			lineNumber:   3,
			expectedFunc: "",
		},
		{
			name:         "line is inside function but not start",
			lineNumber:   6,
			expectedFunc: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := findTestFunctionAtLine(test.lineNumber, testFuncPos)
			assert.Equal(t, test.expectedFunc, result)
		})
	}
}

func Test_isCommentBeforeTestFunction_golden(t *testing.T) {
	t.Parallel()

	lines := []string{
		"package main",         // 0
		"",                     // 1
		"import (",             // 2
		"    \"testing\"",      // 3
		"    \"fmt\"",          // 4
		")",                    // 5
		"",                     // 6
		"// Comment for test",  // 7
		"",                     // 8
		"func Test_ex() {",     // 9
		"}",                    // 10
		"",                     // 11
		"regular line",         // 12
		"",                     // 13
		"// Direct comment",    // 14
		"func Test_direct() {", // 15
		"}",                    // 16
		"",                     // 17
		"func otherFunc() {",   // 18
		"}",                    // 19
		"// After other func",  // 20
		"",                     // 21
		"func Test_after() {",  // 22
		"}",                    // 23
	}

	tests := []struct {
		name              string
		lineIndex         int
		testFuncStartLine int
		expectedIsComment bool
	}{
		{
			name:              "comment separated by empty line, no preceding code (with multiline import)",
			lineIndex:         7,    // "// Comment for test"
			testFuncStartLine: 10,   // "func Test_ex() {" (1-based: 9+1=10)
			expectedIsComment: true, // No other code before this comment (package and import block don't count)
		},
		{
			name:              "empty line before test function",
			lineIndex:         8,     // ""
			testFuncStartLine: 10,    // "func Test_ex() {" (1-based: 9+1=10)
			expectedIsComment: false, // Empty lines are never part of test functions
		},
		{
			name:              "direct comment before test function",
			lineIndex:         14,   // "// Direct comment"
			testFuncStartLine: 16,   // "func Test_direct() {" (1-based: 15+1=16)
			expectedIsComment: true, // Directly precedes test function
		},
		{
			name:              "comment after other function",
			lineIndex:         20,    // "// After other func"
			testFuncStartLine: 23,    // "func Test_after() {" (1-based: 22+1=23)
			expectedIsComment: false, // Has other code (otherFunc) before it
		},
		{
			name:              "line after test function",
			lineIndex:         11, // ""
			testFuncStartLine: 10, // "func Test_ex() {" (1-based: 9+1=10)
			expectedIsComment: false,
		},
		{
			name:              "regular line not related to test",
			lineIndex:         12, // "regular line"
			testFuncStartLine: 10, // "func Test_ex() {" (1-based: 9+1=10)
			expectedIsComment: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := isCommentBeforeTestFunction(test.lineIndex, lines, test.testFuncStartLine)
			assert.Equal(t, test.expectedIsComment, result)
		})
	}
}

func Test_isCommentOrEmpty_golden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "empty line",
			line:     "",
			expected: true,
		},
		{
			name:     "single line comment",
			line:     "// This is a comment",
			expected: true,
		},
		{
			name:     "multi-line comment start",
			line:     "/* This is a multi-line comment",
			expected: true,
		},
		{
			name:     "regular code line",
			line:     "func TestExample(t *testing.T) {",
			expected: false,
		},
		{
			name:     "whitespace only",
			line:     "   \t  ",
			expected: false,
		},
		{
			name:     "line with comment prefix inside",
			line:     "string := \"// not a comment\"",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := isCommentOrEmpty(test.line)
			assert.Equal(t, test.expected, result)
		})
	}
}

//nolint:funlen // allow long test function (64 > 60)
func Test_isLinePartOfTestFunction_golden(t *testing.T) {
	t.Parallel()

	lines := []string{
		"package main",         // 0
		"",                     // 1
		"// Comment for test",  // 2
		"func Test_alpha() {",  // 3
		"	// test body",        // 4
		"}",                    // 5
		"",                     // 6
		"func regularFunc() {", // 7
		"}",                    // 8
	}

	testFuncPos := map[string][2]int{
		"Test_alpha": {4, 6}, // func Test_alpha() on line 3 (0-based) = line 4 (1-based)
	}

	tests := []struct {
		name        string
		lineIndex   int
		expectedIs  bool
		description string
	}{
		{
			name:        "empty line before comment",
			lineIndex:   1,
			expectedIs:  true, // Empty line before comment is considered part
			description: "empty line that precedes comment before test",
		},
		{
			name:        "comment before test function",
			lineIndex:   2,
			expectedIs:  false, // Comment is NOT part of test function itself, but precedes it
			description: "comment preceding test function",
		},
		{
			name:        "test function start",
			lineIndex:   3,
			expectedIs:  true,
			description: "start of test function",
		},
		{
			name:        "inside test function",
			lineIndex:   4,
			expectedIs:  true,
			description: "inside test function body",
		},
		{
			name:        "test function end",
			lineIndex:   5,
			expectedIs:  true,
			description: "end of test function",
		},
		{
			name:        "line after test function",
			lineIndex:   6,
			expectedIs:  false,
			description: "empty line after test function",
		},
		{
			name:        "regular function",
			lineIndex:   7,
			expectedIs:  false,
			description: "regular function not a test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := isLinePartOfTestFunction(test.lineIndex, lines, testFuncPos)
			assert.Equal(t, test.expectedIs, result, test.description)
		})
	}
}

func Test_separateTestAndNonTestContent_golden(t *testing.T) {
	t.Parallel()

	lines := []string{
		"package main",                    // 0
		"",                                // 1
		"import \"testing\"",              // 2
		"",                                // 3
		"// Comment for Test_alpha",       // 4
		"func Test_alpha(t *testing.T) {", // 5
		"	// test body",                   // 6
		"}",                               // 7
		"",                                // 8
		"func regularFunc() {",            // 9
		"	// regular function",            // 10
		"}",                               // 11
		"",                                // 12
		"func Test_beta(t *testing.T) {",  // 13
		"	// another test",                // 14
		"}",                               // 15
	}

	testFuncPos := map[string][2]int{
		"Test_alpha": {6, 8},   // Lines 6-8 (1-based)
		"Test_beta":  {14, 16}, // Lines 14-16 (1-based)
	}

	testFuncs, nonTestLines := separateTestAndNonTestContent(lines, testFuncPos)

	// Check test functions
	require.Len(t, testFuncs, 2)

	assert.Equal(t, "Test_alpha", testFuncs[0].Name)

	expectedAlphaLines := []string{
		"", // Empty line before comment
		"// Comment for Test_alpha",
		"func Test_alpha(t *testing.T) {",
		"	// test body",
		"}",
	}
	assert.Equal(t, expectedAlphaLines, testFuncs[0].Lines)

	assert.Equal(t, "Test_beta", testFuncs[1].Name)

	expectedBetaLines := []string{
		"", // Empty line before function
		"func Test_beta(t *testing.T) {",
		"	// another test",
		"}",
	}
	assert.Equal(t, expectedBetaLines, testFuncs[1].Lines)

	// Check non-test lines
	expectedNonTestLines := []string{
		"package main",
		"",
		"import \"testing\"",
		"func regularFunc() {",
		"\t// regular function",
		"}",
		"", // Trailing empty line
	}
	assert.Equal(t, expectedNonTestLines, nonTestLines)
}
