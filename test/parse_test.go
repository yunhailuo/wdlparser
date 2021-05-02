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
	for _, wdl := range result.Imports {
		k := wdl.GetName()
		if wdl.GetAlias() != "" {
			k = wdl.GetAlias()
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedInput := []*wdlparser.Object{
		wdlparser.NewObject(
			50, 65, wdlparser.Ipt, "input_str", "String", "",
		),
		wdlparser.NewObject(
			75, 94, wdlparser.Ipt, "input_file_path", "File", "",
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedPrivateDecl := []*wdlparser.Object{
		wdlparser.NewObject(
			47, 64, wdlparser.Dcl, "s", "String", `"Hello"`,
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedFirstCall := wdlparser.NewCall(39, 150, "Greeting")
	expectedFirstCall.Inputs = []*wdlparser.Object{
		wdlparser.NewObject(
			91, 113, wdlparser.Ipt, "first_name", "", "first_name",
		),
		wdlparser.NewObject(
			128, 144, wdlparser.Ipt, "last_name", "", `"Luo"`,
		),
	}
	expectedSecondCall := wdlparser.NewCall(156, 213, "Goodbye")
	expectedSecondCall.Inputs = []*wdlparser.Object{
		wdlparser.NewObject(
			190, 210, wdlparser.Ipt, "first_name", "", `"Yunhai"`,
		),
	}
	expectedOutput := []*wdlparser.Call{expectedFirstCall, expectedSecondCall}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultOutput := result.Workflow.Calls
	if !reflect.DeepEqual(resultOutput[0].Inputs, expectedOutput[0].Inputs) {
		t.Errorf(
			"Found inputs for the first call %v, expect %v",
			resultOutput[0].Inputs, expectedOutput[0].Inputs,
		)
	}
	if !reflect.DeepEqual(resultOutput[1].Inputs, expectedOutput[1].Inputs) {
		t.Errorf(
			"Found inputs for the first call %v, expect %v",
			resultOutput[1].Inputs, expectedOutput[1].Inputs,
		)
	}
}

func TestWorkflowOutput(t *testing.T) {
	inputPath := "testdata/workflow_output.wdl"
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedOutput := []*wdlparser.Object{
		wdlparser.NewObject(
			52, 87, wdlparser.Opt, "output_file", "File", `"/Path/to/output"`,
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedMeta := map[string]*wdlparser.Object{
		"author": wdlparser.NewObject(
			48, 67, wdlparser.Mtd, "author", "", `"Yunhai Luo"`,
		),
		"version": wdlparser.NewObject(
			77, 88, wdlparser.Mtd, "version", "", "1.1",
		),
		"for": wdlparser.NewObject(
			98, 112, wdlparser.Mtd, "for", "", `"workflow"`,
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedParameterMeta := map[string]*wdlparser.Object{
		"name": wdlparser.NewObject(
			67, 129, wdlparser.Pmt, "name", "", `{help:"A name for workflow input"}`,
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedInput := []*wdlparser.Object{
		wdlparser.NewObject(
			46, 66, wdlparser.Ipt, "name", "String", `"World"`,
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedPrivateDecl := []*wdlparser.Object{
		wdlparser.NewObject(
			43, 60, wdlparser.Dcl, "s", "String", `"Hello"`,
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
	result, err := wdlparser.Antlr4Parse(inputPath)
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedOutput := []*wdlparser.Object{
		wdlparser.NewObject(
			47, 73, wdlparser.Opt, "output_file", "File", "stdout()",
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedRuntime := map[string]*wdlparser.Object{
		"container": wdlparser.NewObject(
			50, 75, wdlparser.Rnt, "container", "", `"ubuntu:latest"`,
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedMeta := map[string]*wdlparser.Object{
		"author": wdlparser.NewObject(
			44, 63, wdlparser.Mtd, "author", "", `"Yunhai Luo"`,
		),
		"version": wdlparser.NewObject(
			73, 84, wdlparser.Mtd, "version", "", "1.1",
		),
		"for": wdlparser.NewObject(
			94, 104, wdlparser.Mtd, "for", "", `"task"`,
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
	result, err := wdlparser.Antlr4Parse(inputPath)
	expectedParameterMeta := map[string]*wdlparser.Object{
		"name": wdlparser.NewObject(
			63, 122, wdlparser.Pmt, "name", "", `{help:"One name as task input"}`,
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
