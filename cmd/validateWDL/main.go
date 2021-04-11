package main

import (
	"fmt"

	wdlparser "github.com/yunhailuo/wdlparser/pkg"
)

func main() {
	test := [1]string{"../../test/testdata/test.wdl"}
	for _, path := range test {
		// for _, path := range os.Args[1:] {
		_, err := wdlparser.Antlr4Parse(path)
		if err != nil {
			fmt.Printf(
				"Invalid WDL (%q): %d syntax errors found.\n", path, len(err),
			)
		} else {
			fmt.Printf("WDL (%q) is valid.\n", path)
		}
	}
}
