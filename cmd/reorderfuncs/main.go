// Package main provides a command-line tool to reorder test functions in Go source files.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	reorderfuncs "github.com/KEINOS/go-ReOrderFuncs"
)

var errUsage = errors.New(`usage: reorderfuncs <input file> [<output file>]`)

//nolint:gochecknoglobals // osExit and exitOnErr are for mocking in tests
var (
	// osExit is a copy of os.Exit to allow mocking in tests.
	osExit = os.Exit
	// exitOnErr is a func variable to allow mocking os.Exit in tests (monkey patching).
	exitOnErr = func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			osExit(1)
		}
	}
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 || flag.NArg() > 2 {
		exitOnErr(fmt.Errorf("missing or too many arguments\n\n%w", errUsage))
	}

	pathInput := flag.Arg(0)
	pathOutput := pathInput

	if flag.NArg() > 1 {
		pathOutput = flag.Arg(1)
	}

	err := reorderfuncs.Exec(pathInput, pathOutput)
	if err != nil {
		exitOnErr(err)
	}
}
