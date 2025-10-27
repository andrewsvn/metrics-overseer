package noragequit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "noRageQuit", // valid go identifier
	Doc:  "check for panics and abnormal exits outside main method",
	Run:  nrqRun,
}

var excludedPkgs string

func init() {
	Analyzer.Flags.StringVar(&excludedPkgs, "ep", "", "comma-separated list of excluded packages")
}

func nrqRun(pass *analysis.Pass) (interface{}, error) {
	v := NewRageQuitVisitor(excludedPkgs)
	for _, f := range pass.Files {
		if !v.ProcessNextFile(f) {
			continue
		}
		ast.Inspect(f, v.InspectNode)
	}
	v.Report(pass)
	return nil, nil
}
