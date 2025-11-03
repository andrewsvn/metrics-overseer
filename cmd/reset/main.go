package main

import (
	"log"
	"os"

	"github.com/andrewsvn/metrics-overseer/internal/config/resetcfg"
	"golang.org/x/tools/go/packages"
)

func main() {
	_ = os.Chdir(resetcfg.GetConfig().RootDir)

	resetCfg := packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  ".",
	}
	pkgs, err := packages.Load(&resetCfg, "./...")
	if err != nil {
		log.Fatal(err)
	}

	for _, pkg := range pkgs {
		err := ProcessPackage(pkg)
		if err != nil {
			log.Fatal(err)
		}
	}
}
