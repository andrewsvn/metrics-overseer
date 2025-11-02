package main

import (
	"log"
	"os"

	"golang.org/x/tools/go/packages"
)

const (
	rootPath string = `../..`
)

func main() {
	_ = os.Chdir(rootPath)

	cfg := packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  ".",
	}
	pkgs, err := packages.Load(&cfg, "./...")
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
