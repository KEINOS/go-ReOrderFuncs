package main

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // due to monkey patching global variables
func Test_main(t *testing.T) {
	// Backup and defer restore
	originalExitOnErr := exitOnErr

	defer func() { exitOnErr = originalExitOnErr }()

	originalOsArgs := os.Args

	defer func() { os.Args = originalOsArgs }()

	// Replace os.Exit to panic for test
	var capturedErr error

	exitOnErr = func(err error) {
		capturedErr = err
		panic(fmt.Errorf("os.Exit called with error: %w", err))
	}

	tests := []struct {
		name         string
		args         []string
		expectErrMsg string
	}{
		{
			name:         "no arguments",
			args:         []string{"test_name"},
			expectErrMsg: "missing or too many arguments",
		},
		{
			name:         "too many arguments",
			args:         []string{"test_name", "arg1", "arg2", "arg3"},
			expectErrMsg: "missing or too many arguments",
		},
		{
			name:         "non-existent input file",
			args:         []string{"test_name", "non_existent_file.go"},
			expectErrMsg: "open non_existent_file.go: no such file or directory",
		},
		{
			name:         "empty paths",
			args:         []string{"test_name", "", ""},
			expectErrMsg: "open : no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Args = test.args

			require.Panics(t, func() {
				main()
			}, "expected os.Exit to be called")

			require.Contains(t, capturedErr.Error(), test.expectErrMsg,
				"missing expected error message")
		})
	}
}

//nolint:paralleltest // due to monkey patching global variables
func Test_exitOnErr(t *testing.T) {
	originalOsExit := osExit

	defer func() { osExit = originalOsExit }()

	var exitedWithCode int

	osExit = func(code int) {
		exitedWithCode = code
	}

	exitOnErr(errors.New("test error")) //nolint:err113 // allow dynamic error for test

	require.Equal(t, 1, exitedWithCode,
		"expected os.Exit to be called with code 1")
}
