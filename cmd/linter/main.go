package main

import (
	"github.com/andrewsvn/metrics-overseer/cmd/linter/noragequit"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(noragequit.NewAnalyzer())
}
