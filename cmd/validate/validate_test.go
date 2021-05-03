package main

import (
	"bytes"
	"log"
	"os"
	"regexp"
	"testing"
)

func TestCLIvalidate(t *testing.T) {
	buf := new(bytes.Buffer)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	log.SetOutput(buf)
	var tests = []struct {
		path    string
		pattern string
	}{
		{
			"../../pkg/testdata/version1_1.wdl",
			`[0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}` +
				` WDL \("../../pkg/testdata/version1_1.wdl"\) is valid.\n`,
		},
	}
	for _, testcase := range tests {
		os.Args = append([]string{"./validate", "-wdl"}, testcase.path)
		main()
		out := buf.String()
		matched, err := regexp.MatchString(testcase.pattern, out)
		if err != nil {
			t.Fatal("Expected pattern did not compile:", err)
		}
		if !matched {
			t.Errorf(
				"Stdout should match %q is %q",
				testcase.pattern,
				out,
			)
		}
	}
}
