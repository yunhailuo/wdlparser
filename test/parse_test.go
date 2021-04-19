package wdlparser

import (
	"testing"

	wdlparser "github.com/yunhailuo/wdlparser/pkg"
)

func TestBadWDL(t *testing.T) {
	cases := []struct {
		in         string
		errorCount int
	}{
		{"testdata/9errors.wdl", 9},
	}

	for _, c := range cases {
		_, err := wdlparser.Antlr4Parse(c.in)
		if len(err) != c.errorCount {
			t.Errorf(
				"Found %d errors in %q, expect %d",
				len(err), c.in, c.errorCount,
			)
		}
	}
}

func TestVersion(t *testing.T) {
	inputPath := "testdata/hello.wdl"
	expectedVersion := "1.1"
	result, err := wdlparser.Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	if result.Version != expectedVersion {
		t.Errorf(
			"Got version %q, expect version %q",
			result.Version, expectedVersion,
		)
	}
}

func TestImport(t *testing.T) {
	inputPath := "testdata/imports.wdl"
	expectedImports := map[string]string{
		"9errors":  "9errors.wdl",
		"analysis": "http://example.com/lib/analysis_tasks",
		"stdlib":   "https://example.com/lib/stdlib.wdl",
	}
	result, err := wdlparser.Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultImports := result.GetImports()
	if len(resultImports) != len(expectedImports) {
		t.Errorf(
			"Found %d imports in %q, expect %d",
			len(resultImports), inputPath, len(expectedImports),
		)
	}
	for expectedName, expectedPath := range expectedImports {
		if wdl, ok := resultImports[expectedName]; ok {
			if wdl.Path != expectedPath {
				t.Errorf(
					"Imported %q from URI %q, expect URI %q",
					expectedName, wdl.Path, expectedPath,
				)
			}
		} else {
			t.Errorf("Fail to import %q", expectedName)
		}
	}
}

func TestWorkflow(t *testing.T) {
	inputPath := "testdata/hello.wdl"
	expectedName := "HelloWorld"
	expectedElementCount := 1
	result, err := wdlparser.Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultWorkflows := result.GetWorkflow()
	if len(resultWorkflows) != 1 {
		t.Errorf(
			"Found %d workflow in %q, expect 1 workflow",
			len(resultWorkflows), inputPath,
		)
	}
	var resultWorkflow *wdlparser.Workflow
	for _, w := range resultWorkflows {
		resultWorkflow = w
	}
	if resultWorkflow.GetName() != expectedName {
		t.Errorf(
			"Got workflow %q, expect workflow %q",
			resultWorkflow.GetName(), expectedName,
		)
	}
	if len(resultWorkflow.Elements) != expectedElementCount {
		t.Errorf(
			"Found %d workflow elements, expect %d",
			len(resultWorkflow.Elements), expectedElementCount,
		)
	}
}

func TestTask(t *testing.T) {
	inputPath := "testdata/hello.wdl"
	expectedTaskElementCount := map[string]int{
		"WriteGreeting": 2,
	}
	result, err := wdlparser.Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultTasks := result.GetTask()
	if len(resultTasks) != len(expectedTaskElementCount) {
		t.Errorf(
			"Found %d Task in %q, expect %d",
			len(resultTasks), inputPath, len(expectedTaskElementCount),
		)
	}
	for expectedName, expectedElementCount := range expectedTaskElementCount {
		if task, ok := resultTasks[expectedName]; ok {
			if len(task.Elements) != expectedElementCount {
				t.Errorf(
					"Found %d task elements, expect %d",
					len(task.Elements), expectedElementCount,
				)
			}
		} else {
			t.Errorf("Fail to find task %q", expectedName)
		}
	}
}
