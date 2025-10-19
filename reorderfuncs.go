// Package reorderfuncs provides functionality to reorder test functions in Go source files.
package reorderfuncs

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
)

const (
	// importBlockStart represents the start of a multi-line import block.
	importBlockStart = "import ("
)

// TestFunction represents a test function with its content.
type TestFunction struct {
	Name  string
	Lines []string
}

// ============================================================================
//  Public Functions (ABC Order)
// ============================================================================

// BuildOutputContent constructs the final output content from test functions and non-test lines.
func BuildOutputContent(testFuncs []TestFunction, nonTestLines []string) string {
	// Sort test functions alphabetically
	sort.Slice(testFuncs, func(i, j int) bool {
		return testFuncs[i].Name < testFuncs[j].Name
	})

	// Build output content
	var outputLines []string

	outputLines = append(outputLines, nonTestLines...)

	// Add empty line before test functions if there are any
	if len(testFuncs) > 0 && len(nonTestLines) > 0 {
		// Remove trailing empty lines from nonTestLines
		for len(outputLines) > 0 && strings.TrimSpace(outputLines[len(outputLines)-1]) == "" {
			outputLines = outputLines[:len(outputLines)-1]
		}

		outputLines = append(outputLines, "")
	}

	// Add sorted test functions
	for i, testFunc := range testFuncs {
		if i > 0 {
			outputLines = append(outputLines, "")
		}

		// Remove leading empty lines from function content
		funcLines := testFunc.Lines
		for len(funcLines) > 0 && strings.TrimSpace(funcLines[0]) == "" {
			funcLines = funcLines[1:]
		}

		outputLines = append(outputLines, funcLines...)
	}

	// Join and ensure final newline
	output := strings.Join(outputLines, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return output
}

// Exec reorders test functions in a Go source file alphabetically.
func Exec(pathInput, pathOutput string) error {
	// Parse the Go file
	lines, file, fset, err := ParseGoFile(pathInput)
	if err != nil {
		return err // Error already includes proper context from ParseGoFile
	}

	// Extract test functions and non-test content
	testFuncs, nonTestLines := ExtractTestFunctions(lines, file, fset)

	// Build output content
	output := BuildOutputContent(testFuncs, nonTestLines)

	const defaultFileMode = 0o644

	err = os.WriteFile(pathOutput, []byte(output), defaultFileMode)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// ExtractTestFunctions extracts test functions from source lines using AST information.
func ExtractTestFunctions(lines []string, file *ast.File, fset *token.FileSet) ([]TestFunction, []string) {
	testFuncPos := buildTestFunctionPositions(file, fset)

	return separateTestAndNonTestContent(lines, testFuncPos)
}

// ParseGoFile reads and parses a Go source file, returning lines, AST, and FileSet.
func ParseGoFile(filePath string) ([]string, *ast.File, *token.FileSet, error) {
	// Read the file content
	content, err := os.ReadFile(filePath) //nolint:gosec // Input path is controlled by caller
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read input file: %w", err)
	}

	// Parse the file to get AST information
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")

	return lines, file, fset, nil
}

// ============================================================================
//  Private Functions (ABC Order)
// ============================================================================

// buildTestFunctionPositions creates a map of test function positions from AST.
func buildTestFunctionPositions(file *ast.File, fset *token.FileSet) map[string][2]int {
	testFuncPos := make(map[string][2]int) // name -> [start_line, end_line]

	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(function.Name.Name, "Test") {
			continue
		}

		start := fset.Position(function.Pos()).Line
		end := fset.Position(function.End()).Line
		testFuncPos[function.Name.Name] = [2]int{start, end}
	}

	return testFuncPos
}

// extractTestFunctionWithComments extracts a test function including its preceding comments.
func extractTestFunctionWithComments(
	lines []string,
	funcName string,
	testFuncPos map[string][2]int,
) (TestFunction, int) {
	pos := testFuncPos[funcName]
	startLine := pos[0] - 1 // Convert to 0-based
	endLine := pos[1] - 1   // Convert to 0-based

	commentStart := findCommentStart(lines, startLine)

	var funcLines []string
	for i := commentStart; i <= endLine; i++ {
		funcLines = append(funcLines, lines[i])
	}

	return TestFunction{
		Name:  funcName,
		Lines: funcLines,
	}, endLine
}

// findTestFunctionAtLine checks if a specific line starts a test function.
func findTestFunctionAtLine(lineNumber int, testFuncPos map[string][2]int) string {
	for name, pos := range testFuncPos {
		if lineNumber == pos[0] {
			return name
		}
	}

	return ""
}

// findCommentStart finds the start of comments preceding a function.
func findCommentStart(lines []string, functionStartLine int) int {
	commentStart := functionStartLine

	// Find the start of comments and empty lines (mixed) preceding the function
	for commentStart > 0 {
		prevLine := strings.TrimSpace(lines[commentStart-1])
		if strings.HasPrefix(prevLine, "//") || strings.HasPrefix(prevLine, "/*") || prevLine == "" {
			commentStart--
		} else {
			break
		}
	}

	return commentStart
}

// findCommentEnd finds the end of content following a function (including trailing empty lines).
func findCommentEnd(lines []string, functionEndLine int) int {
	commentEnd := functionEndLine

	// Include trailing empty lines after the function
	for commentEnd < len(lines)-1 {
		nextLine := strings.TrimSpace(lines[commentEnd+1])
		if nextLine == "" {
			commentEnd++
		} else {
			break
		}
	}

	return commentEnd
}

// isCommentBeforeTestFunction checks if a line is a comment preceding a test function.
func isCommentBeforeTestFunction(lineIndex int, lines []string, testFuncStartLine int) bool {
	// The testFuncStartLine parameter is expected to be 1-based (AST positions).
	// Convert to 0-based index for comparing with lineIndex.
	if testFuncStartLine <= 0 {
		return false
	}

	testStartIdx := testFuncStartLine - 1

	if lineIndex >= testStartIdx {
		return false
	}

	trimmed := strings.TrimSpace(lines[lineIndex])
	if !isCommentOrEmpty(trimmed) {
		return false
	}

	// Empty lines are never part of test functions
	if trimmed == "" {
		return false
	}

	// Check if this comment is part of the test function:
	// 1. Comment directly precedes the test function (no empty line between)
	if lineIndex+1 == testStartIdx {
		return true
	}

	// 2. Comment is separated by empty lines but has no other code before it
	// Check if there are only empty lines between this comment and the test function
	for i := lineIndex + 1; i < testStartIdx; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return false
		}
	}

	// Check if there's any actual code (not package/import) before this comment
	// Scan backwards to find any non-boilerplate code
	for idx := lineIndex - 1; idx >= 0; idx-- {
		if hasActualCodeAt(lines, idx) {
			return false // Found actual code before this comment
		}
	}

	// No other code found before this comment, so it belongs to the test function
	return true
}

// hasActualCodeAt checks if there is actual (non-boilerplate) code at the given line index.
func hasActualCodeAt(lines []string, idx int) bool {
	line := strings.TrimSpace(lines[idx])

	if isBoilerplateCode(line) {
		return false
	}

	return handleSpecialCases(lines, idx, line)
}

// isBoilerplateCode checks if a line is boilerplate code that should be ignored.
func isBoilerplateCode(line string) bool {
	switch {
	case line == "":
		return true // Skip empty lines
	case strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*"):
		return true // Skip other comments
	case strings.HasPrefix(line, "package "):
		return true // Package declarations don't count as "other code"
	case strings.HasPrefix(line, "import "):
		return true // Single-line imports don't count as "other code"
	case line == importBlockStart:
		return true // Skip entire import block
	default:
		return false
	}
}

// handleSpecialCases handles special cases for import blocks and quoted content.
func handleSpecialCases(lines []string, idx int, line string) bool {
	switch {
	case line == ")":
		// This might be end of import block, skip back to find start
		if isEndOfImportBlock(lines, idx) {
			return false
		}

		return true // Found actual code
	case strings.Contains(line, "\""):
		// Check if this line contains import content
		if isInImportBlock(lines, idx) {
			return false // Skip import content
		}

		return true // Found actual code
	default:
		return true // Found actual code before this comment
	}
}

// isEndOfImportBlock checks if a closing parenthesis is the end of an import block.
func isEndOfImportBlock(lines []string, idx int) bool {
	for innerIdx := idx - 1; innerIdx >= 0; innerIdx-- {
		prevLine := strings.TrimSpace(lines[innerIdx])

		switch {
		case prevLine == "" || strings.HasPrefix(prevLine, "//"):
			continue
		case strings.Contains(prevLine, "\""):
			continue // Import content
		case prevLine == importBlockStart:
			return true
		case strings.HasPrefix(prevLine, "package "):
			return false
		default:
			return false
		}
	}

	return false
}

// isInImportBlock checks if a line with quotes is inside an import block.
func isInImportBlock(lines []string, idx int) bool {
	for innerIdx := idx - 1; innerIdx >= 0; innerIdx-- {
		prevLine := strings.TrimSpace(lines[innerIdx])

		switch {
		case prevLine == "" || strings.HasPrefix(prevLine, "//"):
			continue
		case strings.Contains(prevLine, "\""):
			continue // Other import content
		case prevLine == importBlockStart:
			return true
		case strings.HasPrefix(prevLine, "package "):
			return false
		default:
			return false
		}
	}

	return false
}

// isCommentOrEmpty checks if a line is a comment or empty.
func isCommentOrEmpty(line string) bool {
	return line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*")
}

// isLinePartOfTestFunction checks if a line is part of any test function.
func isLinePartOfTestFunction(lineIndex int, lines []string, testFuncPos map[string][2]int) bool {
	for _, pos := range testFuncPos {
		// pos contains 1-based start and end line numbers (from token.FileSet).
		startLine := pos[0]
		endLine := pos[1]

		// Convert to 0-based indices for comparison with lineIndex
		startIdx := startLine - 1
		endIdx := endLine - 1

		// Check if line is within test function range
		if lineIndex >= startIdx && lineIndex <= endIdx {
			return true
		}

		// Check if this is an empty line that precedes a comment before test function
		if isEmptyLinePrecedingComment(lineIndex, lines, startIdx) {
			return true
		}
	}

	return false
}

// isEmptyLinePrecedingComment checks if an empty line precedes a comment before a test function.
func isEmptyLinePrecedingComment(lineIndex int, lines []string, startIdx int) bool {
	// Line must be before the test function
	if lineIndex >= startIdx {
		return false
	}

	// Line must be empty
	line := strings.TrimSpace(lines[lineIndex])
	if line != "" {
		return false
	}

	// Check if there's a comment between this empty line and the test function
	return hasCommentBeforeTest(lineIndex, lines, startIdx)
}

// hasCommentBeforeTest checks if there's a comment between an empty line and a test function.
func hasCommentBeforeTest(lineIndex int, lines []string, startIdx int) bool {
	for forwardIndex := lineIndex + 1; forwardIndex < startIdx; forwardIndex++ {
		nextLine := strings.TrimSpace(lines[forwardIndex])
		if nextLine == "" {
			continue // Skip empty lines
		}

		if !strings.HasPrefix(nextLine, "//") && !strings.HasPrefix(nextLine, "/*") {
			break // Found non-comment content
		}

		// Found comment, check if it's the last non-empty content before test function
		if isLastContentBeforeTest(forwardIndex, lines, startIdx) {
			return true // Empty line precedes comment before test
		}

		break // Found non-comment content
	}

	return false
}

// isLastContentBeforeTest checks if a comment is the last content before a test function.
func isLastContentBeforeTest(commentIndex int, lines []string, startIdx int) bool {
	for innerIdx := commentIndex + 1; innerIdx < startIdx; innerIdx++ {
		if strings.TrimSpace(lines[innerIdx]) != "" {
			return false
		}
	}

	return true
}

// funcPos represents the position of a function in the source code.
type funcPos struct {
	name      string
	startLine int
	endLine   int
}

// separateTestAndNonTestContent processes lines to separate test functions from other content.
func separateTestAndNonTestContent(lines []string, testFuncPos map[string][2]int) ([]TestFunction, []string) {
	sortedFuncs := createSortedFuncPositions(testFuncPos)
	processedLines := markProcessedLines(lines, sortedFuncs)
	testFuncs := extractAllTestFunctions(lines, sortedFuncs, testFuncPos)
	nonTestLines := collectNonTestLines(lines, processedLines)

	return testFuncs, nonTestLines
}

// createSortedFuncPositions creates a sorted list of test function positions.
func createSortedFuncPositions(testFuncPos map[string][2]int) []funcPos {
	sortedFuncs := make([]funcPos, 0, len(testFuncPos))
	for name, pos := range testFuncPos {
		sortedFuncs = append(sortedFuncs, funcPos{
			name:      name,
			startLine: pos[0] - 1, // Convert to 0-based
			endLine:   pos[1] - 1, // Convert to 0-based
		})
	}

	// Sort by line number to ensure deterministic order
	for i := 0; i < len(sortedFuncs); i++ {
		for j := i + 1; j < len(sortedFuncs); j++ {
			if sortedFuncs[i].startLine > sortedFuncs[j].startLine {
				sortedFuncs[i], sortedFuncs[j] = sortedFuncs[j], sortedFuncs[i]
			}
		}
	}

	return sortedFuncs
}

// markProcessedLines marks all lines that are part of test functions and their comments.
func markProcessedLines(lines []string, sortedFuncs []funcPos) map[int]bool {
	processedLines := make(map[int]bool)

	for _, funcInfo := range sortedFuncs {
		startLine := funcInfo.startLine
		endLine := funcInfo.endLine

		// Ensure we don't go out of bounds
		if startLine >= 0 && startLine < len(lines) && endLine >= 0 && endLine < len(lines) {
			commentStart := findCommentStart(lines, startLine)
			commentEnd := findCommentEnd(lines, endLine)

			// Mark all lines from comment start to comment end as processed
			for i := commentStart; i <= commentEnd && i < len(lines); i++ {
				processedLines[i] = true
			}
		}
	}

	return processedLines
}

// extractAllTestFunctions extracts all test functions using the sorted positions.
func extractAllTestFunctions(lines []string, sortedFuncs []funcPos, testFuncPos map[string][2]int) []TestFunction {
	var testFuncs []TestFunction

	for _, funcInfo := range sortedFuncs {
		startLine := funcInfo.startLine
		if startLine >= 0 && startLine < len(lines) {
			testFunc, _ := extractTestFunctionWithComments(lines, funcInfo.name, testFuncPos)
			testFuncs = append(testFuncs, testFunc)
		}
	}

	return testFuncs
}

// collectNonTestLines collects all lines that haven't been processed as test functions.
func collectNonTestLines(lines []string, processedLines map[int]bool) []string {
	var nonTestLines []string

	for i, line := range lines {
		if !processedLines[i] {
			nonTestLines = append(nonTestLines, line)
		}
	}

	// Add trailing empty line if there are any non-test lines
	if len(nonTestLines) > 0 {
		nonTestLines = append(nonTestLines, "")
	}

	return nonTestLines
}
