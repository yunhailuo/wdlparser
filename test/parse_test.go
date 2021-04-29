package wdlparser_test

import (
	"reflect"
	"testing"

	wdlparser "github.com/yunhailuo/wdlparser/pkg"
)

func TestVersion(t *testing.T) {
	inputPath := "testdata/version1_1.wdl"
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
	inputPath := "testdata/import.wdl"
	expectedImportPaths := map[string]string{
		"test":     "test.wdl",
		"analysis": "http://example.com/lib/analysis_tasks",
		"stdlib":   "https://example.com/lib/stdlib.wdl",
	}
	result, err := wdlparser.Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultImportPaths := make(map[string]string)
	for k, wdl := range result.GetImports() {
		resultImportPaths[k] = wdl.Path
	}
	if !reflect.DeepEqual(resultImportPaths, expectedImportPaths) {
		t.Errorf(
			"Found imports %v, expect %v",
			resultImportPaths, expectedImportPaths,
		)
	}
}

func TestWorkflowInput(t *testing.T) {
	inputPath := "testdata/workflow_input.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedInput := map[string]wdlparser.Decl{
		"input_str": wdlparser.NewObject(
			wdlparser.Ipt, "input_str", "String", "",
		),
		"input_file_path": wdlparser.NewObject(
			wdlparser.Ipt, "input_file_path", "File", "",
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultInput := result.GetWorkflow()["Input"].GetInput()
	if !reflect.DeepEqual(resultInput, expectedInput) {
		t.Errorf(
			"Found workflow input %v, expect %v",
			resultInput, expectedInput,
		)
	}
}

func TestWorkflowOutput(t *testing.T) {
	inputPath := "testdata/workflow_output.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedOutput := map[string]wdlparser.Decl{
		"output_file": wdlparser.NewObject(
			wdlparser.Opt, "output_file", "File", `"/Path/to/output"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultOutput := result.GetWorkflow()["Output"].GetOutput()
	if !reflect.DeepEqual(resultOutput, expectedOutput) {
		t.Errorf(
			"Found workflow output %v, expect %v",
			resultOutput, expectedOutput,
		)
	}
}

func TestWorkflowMeta(t *testing.T) {
	inputPath := "testdata/workflow_meta.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedMeta := map[string]wdlparser.Decl{
		"author": wdlparser.NewObject(
			wdlparser.Mtd, "author", "", `"Yunhai Luo"`,
		),
		"version": wdlparser.NewObject(
			wdlparser.Mtd, "version", "", "1.1",
		),
		"for": wdlparser.NewObject(
			wdlparser.Mtd, "for", "", `"workflow"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultMeta := result.GetWorkflow()["Meta"].GetMetadata()
	if !reflect.DeepEqual(resultMeta, expectedMeta) {
		t.Errorf(
			"Found workflow metadata %v, expect %v",
			resultMeta, expectedMeta,
		)
	}
}

func TestWorkflowParameterMeta(t *testing.T) {
	inputPath := "testdata/workflow_parameter_meta.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedParameterMeta := map[string]wdlparser.Decl{
		"name": wdlparser.NewObject(
			wdlparser.Pmt, "name", "", `{help:"A name for workflow input"}`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultParameterMeta := result.GetWorkflow()["ParameterMeta"].GetParameterMetadata()
	if !reflect.DeepEqual(resultParameterMeta, expectedParameterMeta) {
		t.Errorf(
			"Found workflow parameter metadata %v, expect %v",
			resultParameterMeta, expectedParameterMeta,
		)
	}
}

func TestTaskInput(t *testing.T) {
	inputPath := "testdata/task_input.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedInput := map[string]wdlparser.Decl{
		"name": wdlparser.NewObject(
			wdlparser.Ipt, "name", "String", `"World"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultInput := result.GetTask()["Input"].GetInput()
	if !reflect.DeepEqual(resultInput, expectedInput) {
		t.Errorf(
			"Found task input %v, expect %v",
			resultInput, expectedInput,
		)
	}
}

func TestTaskPrivateDeclaration(t *testing.T) {
	inputPath := "testdata/task_private_declaration.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedPrivateDecl := map[string]wdlparser.Decl{
		"s": wdlparser.NewObject(
			wdlparser.Dcl, "s", "String", `"Hello"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultPrivateDecl := result.GetTask()["PrivateDeclaration"].GetPrivateDecl()
	if !reflect.DeepEqual(resultPrivateDecl, expectedPrivateDecl) {
		t.Errorf(
			"Found task private declaration %v, expect %v",
			resultPrivateDecl, expectedPrivateDecl,
		)
	}
}

func TestTaskCommand(t *testing.T) {
	inputPath := "testdata/task_command.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedCommand := []string{"\n        echo \"Hello world\"\n    "}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultCommand := result.GetTask()["Command"].Command
	if !reflect.DeepEqual(resultCommand, expectedCommand) {
		t.Errorf(
			"Found task command %v, expect %v",
			resultCommand, expectedCommand,
		)
	}
}

func TestTaskOutput(t *testing.T) {
	inputPath := "testdata/task_output.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedOutput := map[string]wdlparser.Decl{
		"output_file": wdlparser.NewObject(
			wdlparser.Opt, "output_file", "File", "stdout()",
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultOutput := result.GetTask()["Output"].GetOutput()
	if !reflect.DeepEqual(resultOutput, expectedOutput) {
		t.Errorf(
			"Found task output %v, expect %v",
			resultOutput, expectedOutput,
		)
	}
}

func TestTaskRuntime(t *testing.T) {
	inputPath := "testdata/task_runtime.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedRuntime := map[string]wdlparser.Decl{
		"container": wdlparser.NewObject(
			wdlparser.Rnt, "container", "", `"ubuntu:latest"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultRuntime := result.GetTask()["Runtime"].GetRuntime()
	if !reflect.DeepEqual(resultRuntime, expectedRuntime) {
		t.Errorf(
			"Found task runtime %v, expect %v",
			resultRuntime, expectedRuntime,
		)
	}
}

func TestTaskMeta(t *testing.T) {
	inputPath := "testdata/task_meta.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedMeta := map[string]wdlparser.Decl{
		"author": wdlparser.NewObject(
			wdlparser.Mtd, "author", "", `"Yunhai Luo"`,
		),
		"version": wdlparser.NewObject(
			wdlparser.Mtd, "version", "", "1.1",
		),
		"for": wdlparser.NewObject(
			wdlparser.Mtd, "for", "", `"task"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultMeta := result.GetTask()["Meta"].GetMetadata()
	if !reflect.DeepEqual(resultMeta, expectedMeta) {
		t.Errorf(
			"Found task metadata %v, expect %v",
			resultMeta, expectedMeta,
		)
	}
}

func TestTaskParameterMeta(t *testing.T) {
	inputPath := "testdata/task_parameter_meta.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedParameterMeta := map[string]wdlparser.Decl{
		"name": wdlparser.NewObject(
			wdlparser.Pmt, "name", "", `{help:"One name as task input"}`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultParameterMeta := result.GetTask()["ParameterMeta"].GetParameterMetadata()
	if !reflect.DeepEqual(resultParameterMeta, expectedParameterMeta) {
		t.Errorf(
			"Found task parameter metadata %v, expect %v",
			resultParameterMeta, expectedParameterMeta,
		)
	}
}
