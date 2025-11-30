package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestExitCheckAnalyzer(t *testing.T) {
	// Run the analyzer on test packages
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, ExitCheckAnalyzer, "a", "mainpkg")
}
