package testdata

import (
	"fmt"
	"testing"
)

// Global constant
const GlobalConstant = "test"

// Regular variable
var globalVar = "initial"

// Helper function before tests
func helperFunction(input string) string {
	return fmt.Sprintf("processed: %s", input)
}

// Test function with complex comments
// It includes multiple scenarios and edge cases

// Regular struct definition
type TestStruct struct {
	Field1 string
	Field2 int
}

// Method on struct (not a test)
func (ts *TestStruct) Method() string {
	return fmt.Sprintf("%s-%d", ts.Field1, ts.Field2)
}
// Another test function

// Interface definition
type TestInterface interface {
	DoSomething() error
}

// Implementation of interface
type TestImplementation struct{}

func (ti *TestImplementation) DoSomething() error {
	return nil
}
// Test function with setup/teardown pattern

// Another regular function
func anotherHelper(x, y int) int {
	return x + y
}
// Final test function

// Another test function
func Test_alpha(t *testing.T) {
	ts := &TestStruct{"alpha", 1}
	result := ts.Method()
	if result != "alpha-1" {
		t.Errorf("expected 'alpha-1', got %s", result)
	}
}

// Test function with setup/teardown pattern
func Test_beta(t *testing.T) {
	// Setup
	impl := &TestImplementation{}

	// Test
	err := impl.DoSomething()

	// Verify
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// Final test function
func Test_gamma(t *testing.T) {
	result := anotherHelper(2, 3)
	if result != 5 {
		t.Errorf("expected 5, got %d", result)
	}
}

// Test function with complex comments
// This test validates the zulu functionality
// It includes multiple scenarios and edge cases
func Test_zulu(t *testing.T) {
	result := helperFunction("zulu")
	if result != "processed: zulu" {
		t.Errorf("expected 'processed: zulu', got %s", result)
	}
}
