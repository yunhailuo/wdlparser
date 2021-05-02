package wdlparser

import (
	"reflect"
	"testing"
)

func TestVersion(t *testing.T) {
	inputPath := "testdata/version1_1.wdl"
	expectedVersion := "1.1"
	result, err := Antlr4Parse(inputPath)
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
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultImportPaths := make(map[string]string)
	for _, wdl := range result.Imports {
		k := wdl.getName()
		if wdl.getAlias() != "" {
			k = wdl.getAlias()
		}
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
	result, err := Antlr4Parse(inputPath)
	expectedInput := []*object{
		newObject(
			50, 65, ipt, "input_str", "String", "",
		),
		newObject(
			75, 94, ipt, "input_file_path", "File", "",
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultInput := result.Workflow.Inputs
	if !reflect.DeepEqual(resultInput, expectedInput) {
		t.Errorf(
			"Found workflow input %v, expect %v",
			resultInput, expectedInput,
		)
	}
}

func TestWorkflowPrivateDeclaration(t *testing.T) {
	inputPath := "testdata/workflow_private_declaration.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedPrivateDecl := []*object{
		newObject(
			47, 64, dcl, "s", "String", `"Hello"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultPrivateDecl := result.Workflow.PrvtDecls
	if !reflect.DeepEqual(resultPrivateDecl, expectedPrivateDecl) {
		t.Errorf(
			"Found workflow private declaration %v, expect %v",
			resultPrivateDecl, expectedPrivateDecl,
		)
	}
}

func TestWorkflowCall(t *testing.T) {
	inputPath := "testdata/workflow_call.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedFirstCall := NewCall(39, 150, "Greeting")
	expectedFirstCall.setAlias("hello")
	expectedFirstCall.Inputs = []*object{
		newObject(
			91, 113, ipt, "first_name", "", "first_name",
		),
		newObject(
			128, 144, ipt, "last_name", "", `"Luo"`,
		),
	}
	expectedSecondCall := NewCall(156, 213, "Goodbye")
	expectedSecondCall.After = "hello"
	expectedSecondCall.Inputs = []*object{
		newObject(
			190, 210, ipt, "first_name", "", `"Yunhai"`,
		),
	}
	expectCalls := []*Call{expectedFirstCall, expectedSecondCall}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultCalls := result.Workflow.Calls
	for _, c := range resultCalls {
		c.setParent(nil)
	}
	if !reflect.DeepEqual(resultCalls, expectCalls) {
		t.Errorf(
			"Found inputs for the first call %v, expect %v",
			resultCalls, expectCalls,
		)
	}
}

func TestWorkflowOutput(t *testing.T) {
	inputPath := "testdata/workflow_output.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedOutput := []*object{
		newObject(
			52, 87, opt, "output_file", "File", `"/Path/to/output"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultOutput := result.Workflow.Outputs
	if !reflect.DeepEqual(resultOutput, expectedOutput) {
		t.Errorf(
			"Found workflow output %v, expect %v",
			resultOutput, expectedOutput,
		)
	}
}

func TestWorkflowMeta(t *testing.T) {
	inputPath := "testdata/workflow_meta.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedMeta := map[string]*object{
		"author": newObject(
			48, 67, mtd, "author", "", `"Yunhai Luo"`,
		),
		"version": newObject(
			77, 88, mtd, "version", "", "1.1",
		),
		"for": newObject(
			98, 112, mtd, "for", "", `"workflow"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultMeta := result.Workflow.Meta
	if !reflect.DeepEqual(resultMeta, expectedMeta) {
		t.Errorf(
			"Found workflow metadata %v, expect %v",
			resultMeta, expectedMeta,
		)
	}
}

func TestWorkflowParameterMeta(t *testing.T) {
	inputPath := "testdata/workflow_parameter_meta.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedParameterMeta := map[string]*object{
		"name": newObject(
			67, 129, pmt, "name", "", `{help:"A name for workflow input"}`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultParameterMeta := result.Workflow.ParameterMeta
	if !reflect.DeepEqual(resultParameterMeta, expectedParameterMeta) {
		t.Errorf(
			"Found workflow parameter metadata %v, expect %v",
			resultParameterMeta, expectedParameterMeta,
		)
	}
}

func TestTaskInput(t *testing.T) {
	inputPath := "testdata/task_input.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedInput := []*object{
		newObject(
			46, 66, ipt, "name", "String", `"World"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultInput := result.Tasks[0].Inputs
	if !reflect.DeepEqual(resultInput, expectedInput) {
		t.Errorf(
			"Found task input %v, expect %v",
			resultInput, expectedInput,
		)
	}
}

func TestTaskPrivateDeclaration(t *testing.T) {
	inputPath := "testdata/task_private_declaration.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedPrivateDecl := []*object{
		newObject(
			43, 60, dcl, "s", "String", `"Hello"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultPrivateDecl := result.Tasks[0].PrvtDecls
	if !reflect.DeepEqual(resultPrivateDecl, expectedPrivateDecl) {
		t.Errorf(
			"Found task private declaration %v, expect %v",
			resultPrivateDecl, expectedPrivateDecl,
		)
	}
}

func TestTaskCommand(t *testing.T) {
	inputPath := "testdata/task_command.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedCommand := []string{
		"\n        echo \"Hello ", "~{world}", "\"\n    ",
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultCommand := result.Tasks[0].Command
	if !reflect.DeepEqual(resultCommand, expectedCommand) {
		t.Errorf(
			"Found task command %v, expect %v",
			resultCommand, expectedCommand,
		)
	}
}

func TestTaskOutput(t *testing.T) {
	inputPath := "testdata/task_output.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedOutput := []*object{
		newObject(
			47, 73, opt, "output_file", "File", "stdout()",
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultOutput := result.Tasks[0].Outputs
	if !reflect.DeepEqual(resultOutput, expectedOutput) {
		t.Errorf(
			"Found task output %v, expect %v",
			resultOutput, expectedOutput,
		)
	}
}

func TestTaskRuntime(t *testing.T) {
	inputPath := "testdata/task_runtime.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedRuntime := map[string]*object{
		"container": newObject(
			50, 75, rnt, "container", "", `"ubuntu:latest"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultRuntime := result.Tasks[0].Runtime
	if !reflect.DeepEqual(resultRuntime, expectedRuntime) {
		t.Errorf(
			"Found task runtime %v, expect %v",
			resultRuntime, expectedRuntime,
		)
	}
}

func TestTaskMeta(t *testing.T) {
	inputPath := "testdata/task_meta.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedMeta := map[string]*object{
		"author": newObject(
			44, 63, mtd, "author", "", `"Yunhai Luo"`,
		),
		"version": newObject(
			73, 84, mtd, "version", "", "1.1",
		),
		"for": newObject(
			94, 104, mtd, "for", "", `"task"`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultMeta := result.Tasks[0].Meta
	if !reflect.DeepEqual(resultMeta, expectedMeta) {
		t.Errorf(
			"Found task metadata %v, expect %v",
			resultMeta, expectedMeta,
		)
	}
}

func TestTaskParameterMeta(t *testing.T) {
	inputPath := "testdata/task_parameter_meta.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedParameterMeta := map[string]*object{
		"name": newObject(
			63, 122, pmt, "name", "", `{help:"One name as task input"}`,
		),
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultParameterMeta := result.Tasks[0].ParameterMeta
	if !reflect.DeepEqual(resultParameterMeta, expectedParameterMeta) {
		t.Errorf(
			"Found task parameter metadata %v, expect %v",
			resultParameterMeta, expectedParameterMeta,
		)
	}
}
