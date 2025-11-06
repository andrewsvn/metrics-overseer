package noragequit

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNoRageQuitAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), NewAnalyzer(), "./...")
}
