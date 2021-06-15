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
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}

	import1 := newImportSpec(13, 29, "test.wdl")
	import1.setParent(result)
	import2 := newImportSpec(31, 88, "http://example.com/lib/analysis_tasks")
	import2.setParent(result)
	import2.alias = "analysis"
	import3 := newImportSpec(90, 216, "https://example.com/lib/stdlib.wdl")
	import3.setParent(result)
	import3.importAliases = map[string]string{
		"Parent":     "Parent2",
		"Child":      "Child2",
		"GrandChild": "GrandChild2",
	}
	expectedImports := []importSpec{*import1, *import2, *import3}
	resultImports := []importSpec{}
	for i := range result.Imports {
		resultImports = append(resultImports, *result.Imports[i])
	}
	if !reflect.DeepEqual(resultImports, expectedImports) {
		t.Errorf(
			"Found imports %v, expect %v",
			resultImports, expectedImports,
		)
	}
}

func TestWorkflowInput(t *testing.T) {
	inputPath := "testdata/workflow_input.wdl"
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}

	input1 := newDecl(50, 65, "input_str", "String", "")
	input1.setParent(result.Workflow.Inputs)
	input2 := newDecl(75, 94, "input_file_path", "File", "")
	input2.setParent(result.Workflow.Inputs)
	expectedInput := []*decl{input1, input2}
	resultInput := result.Workflow.Inputs.decls
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
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}

	prvtDecl1 := newDecl(47, 64, "s", "String", `"Hello"`)
	prvtDecl1.setParent(result.Workflow)
	expectedPrivateDecl := []*decl{prvtDecl1}
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
	expectedFirstCall.alias = "hello"
	expectedFirstCall.Inputs = []*keyValue{
		newKeyValue(91, 113, "first_name", "first_name"),
		newKeyValue(128, 144, "last_name", `"Luo"`),
	}
	expectedSecondCall := NewCall(156, 213, "Goodbye")
	expectedSecondCall.After = "hello"
	expectedSecondCall.Inputs = []*keyValue{
		newKeyValue(190, 210, "first_name", `"Yunhai"`),
	}
	expectCalls := []*Call{expectedFirstCall, expectedSecondCall}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultCalls := result.Workflow.Calls
	for _, c := range expectCalls {
		c.setParent(result.Workflow)
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
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}

	output1 := newDecl(52, 87, "output_file", "File", `"/Path/to/output"`)
	output1.setParent(result.Workflow.Outputs)
	expectedOutput := []*decl{output1}
	resultOutput := result.Workflow.Outputs.decls
	if !reflect.DeepEqual(resultOutput, expectedOutput) {
		t.Errorf(
			"Found workflow output %v, expect %v",
			resultOutput, expectedOutput,
		)
	}
}

func TestWorkflowMeta(t *testing.T) {
	inputPath := "testdata/workflow_meta.wdl"
	expectedMeta := map[string]string{
		"author":  `"Yunhai Luo"`,
		"version": "1.1",
		"for":     `"workflow"`,
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultMeta := result.Workflow.Meta.keyValues
	if !reflect.DeepEqual(resultMeta, expectedMeta) {
		t.Errorf(
			"Found workflow metadata %v, expect %v",
			resultMeta, expectedMeta,
		)
	}
}

func TestWorkflowParameterMeta(t *testing.T) {
	inputPath := "testdata/workflow_parameter_meta.wdl"
	expectedParameterMeta := map[string]string{
		"name": `{help:"A name for workflow input"}`,
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultParameterMeta := result.Workflow.ParameterMeta.keyValues
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
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}

	input1 := newDecl(46, 66, "name", "String", `"World"`)
	input1.setParent(result.Tasks[0].Inputs)
	expectedInput := []*decl{input1}
	resultInput := result.Tasks[0].Inputs.decls
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
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}

	prvtDecl1 := newDecl(43, 60, "s", "String", `"Hello"`)
	prvtDecl1.setParent(result.Tasks[0])
	expectedPrivateDecl := []*decl{prvtDecl1}
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
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}

	output1 := newDecl(47, 73, "output_file", "File", "stdout()")
	output1.setParent(result.Tasks[0].Outputs)
	expectedOutput := []*decl{output1}
	resultOutput := result.Tasks[0].Outputs.decls
	if !reflect.DeepEqual(resultOutput, expectedOutput) {
		t.Errorf(
			"Found task output %v, expect %v",
			resultOutput, expectedOutput,
		)
	}
}

func TestTaskRuntime(t *testing.T) {
	inputPath := "testdata/task_runtime.wdl"
	expectedRuntime := map[string]string{
		"container": `"ubuntu:latest"`,
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultRuntime := result.Tasks[0].Runtime.keyValues
	if !reflect.DeepEqual(resultRuntime, expectedRuntime) {
		t.Errorf(
			"Found task runtime %v, expect %v",
			resultRuntime, expectedRuntime,
		)
	}
}

func TestTaskMeta(t *testing.T) {
	inputPath := "testdata/task_meta.wdl"
	expectedMeta := map[string]string{
		"author":  `"Yunhai Luo"`,
		"version": "1.1",
		"for":     `"task"`,
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultMeta := result.Tasks[0].Meta.keyValues
	if !reflect.DeepEqual(resultMeta, expectedMeta) {
		t.Errorf(
			"Found task metadata %v, expect %v",
			resultMeta, expectedMeta,
		)
	}
}

func TestTaskParameterMeta(t *testing.T) {
	inputPath := "testdata/task_parameter_meta.wdl"
	expectedParameterMeta := map[string]string{
		"name": `{help:"One name as task input"}`,
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultParameterMeta := result.Tasks[0].ParameterMeta.keyValues
	if !reflect.DeepEqual(resultParameterMeta, expectedParameterMeta) {
		t.Errorf(
			"Found task parameter metadata %v, expect %v",
			resultParameterMeta, expectedParameterMeta,
		)
	}
}
