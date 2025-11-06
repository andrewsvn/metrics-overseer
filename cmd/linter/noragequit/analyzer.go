package noragequit

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func NewAnalyzer() *analysis.Analyzer {
	var excludedPkgs string

	analyzer := &analysis.Analyzer{
		Name: "noRageQuit", // valid go identifier
		Doc:  "check for panics and abnormal exits outside main method",
		Run: func(pass *analysis.Pass) (any, error) {
			return nrqRun(pass, excludedPkgs)
		},
	}
	analyzer.Flags.StringVar(&excludedPkgs, "ep", "", "comma-separated list of excluded packages")
	return analyzer
}

func nrqRun(pass *analysis.Pass, excludedPkgs string) (any, error) {
	excludedPkgsList := strings.Split(excludedPkgs, ",")
	for _, pkg := range excludedPkgsList {
		if pass.Pkg.Name() == pkg {
			return nil, nil
		}
	}

	v := NewRageQuitVisitor(pass.TypesInfo)
	for _, f := range pass.Files {
		v.ProcessNextFile(f)
		ast.Inspect(f, v.InspectNode)
	}
	v.Report(pass)
	return nil, nil
}
