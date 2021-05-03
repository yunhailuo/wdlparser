package main

import (
	"flag"
	"log"
	"os"

	wdlparser "github.com/yunhailuo/wdlparser/pkg"
)

func main() {
	var path string
	flag.StringVar(&path, "wdl", "", "path to a WDL document to be validated")
	flag.Parse()

	f, err := os.Stat(path)
	if os.IsNotExist(err) || f.IsDir() {
		log.Printf("%v is not a path to a valid file\n\n", path)
		flag.Usage()
		os.Exit(1)
	}

	_, errs := wdlparser.Antlr4Parse(path)
	if errs != nil {
		log.Printf(
			"Invalid WDL (%q): found %d syntax errors.\n", path, len(errs),
		)
	} else {
		log.Printf("WDL (%q) is valid.\n", path)
	}
}
