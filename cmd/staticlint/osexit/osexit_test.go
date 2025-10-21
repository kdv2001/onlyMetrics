package osexit

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestCheckAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), CheckAnalyzer, "./...")
}
