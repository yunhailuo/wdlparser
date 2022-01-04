package wdlparser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var commonCmpopts = cmp.Options{
	cmp.AllowUnexported(genNode{}, identifier{}, namedNode{}),
	cmpopts.IgnoreFields(genNode{}, "parent"),
}

func TestVersion(t *testing.T) {
	inputPath := "testdata/version1_1.wdl"
	expectedVersion := "1.1"
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	if diff := cmp.Diff(expectedVersion, result.Version); diff != "" {
		t.Errorf("unexpected WDL version:\n%s", diff)
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
	import2 := newImportSpec(31, 88, "http://example.com/lib/analysis_tasks")
	import2.alias = "analysis"
	import3 := newImportSpec(90, 216, "https://example.com/lib/stdlib.wdl")
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
	cmpOptions := append(commonCmpopts, cmp.AllowUnexported(importSpec{}))
	if diff := cmp.Diff(
		expectedImports, resultImports, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected imports:\n%s", diff)
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

	input1 := newDecl(50, 65, "input_str", "String")
	input2 := newDecl(75, 94, "input_file_path", "File")
	expectedInput := []*decl{input1, input2}
	resultInput := result.Workflow.Inputs
	cmpOptions := append(commonCmpopts, cmp.AllowUnexported(decl{}))
	if diff := cmp.Diff(
		expectedInput, resultInput, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected workflow input:\n%s", diff)
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

	expectedPrivateDecl := []*decl{
		{
			genNode:    genNode{start: 47, end: 64},
			identifier: "s",
			typ:        "String",
		},
	}
	resultPrivateDecl := result.Workflow.PrvtDecls
	cmpOptions := append(commonCmpopts, cmp.AllowUnexported(decl{}))
	if diff := cmp.Diff(
		expectedPrivateDecl, resultPrivateDecl, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected workflow private declaration:\n%s", diff)
	}
}

func TestWorkflowCall(t *testing.T) {
	inputPath := "testdata/workflow_call.wdl"
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}

	expectCalls := []*Call{
		{
			namedNode: namedNode{
				genNode: genNode{start: 39, end: 150},
				name:    newIdentifier("Greeting", false),
				alias:   "hello",
			},
			Inputs: map[identifier]*exprRPN{
				newIdentifier("first_name", true): {"first_name"},
				newIdentifier("last_name", true):  {},
			},
		},
		{
			namedNode: namedNode{
				genNode: genNode{start: 156, end: 213},
				name:    newIdentifier("Goodbye", false),
			},
			After: "hello",
			Inputs: map[identifier]*exprRPN{
				newIdentifier("first_name", true): {},
			},
		},
	}
	resultCalls := result.Workflow.Calls
	cmpOptions := append(commonCmpopts, cmp.AllowUnexported(Call{}))
	if diff := cmp.Diff(
		expectCalls, resultCalls, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected workflow calls:\n%s", diff)
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

	expectedOutput := []*decl{
		{
			genNode:    genNode{start: 52, end: 87},
			identifier: "output_file",
			typ:        "File",
		},
	}
	resultOutput := result.Workflow.Outputs
	cmpOptions := append(commonCmpopts, cmp.AllowUnexported(decl{}))
	if diff := cmp.Diff(
		expectedOutput, resultOutput, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected workflow output:\n%s", diff)
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
	resultMeta := result.Workflow.Meta
	if diff := cmp.Diff(expectedMeta, resultMeta); diff != "" {
		t.Errorf("unexpected workflow metadata:\n%s", diff)
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
	resultParameterMeta := result.Workflow.ParameterMeta
	if diff := cmp.Diff(
		expectedParameterMeta, resultParameterMeta,
	); diff != "" {
		t.Errorf("unexpected workflow parameter metadata:\n%s", diff)
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

	expectedInput := []*decl{
		{
			genNode:    genNode{start: 46, end: 66},
			identifier: "name",
			typ:        "String",
		},
	}
	resultInput := result.Tasks[0].Inputs
	cmpOptions := append(commonCmpopts, cmp.AllowUnexported(decl{}))
	if diff := cmp.Diff(
		expectedInput, resultInput, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected task input:\n%s", diff)
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

	expectedPrivateDecl := []*decl{
		{
			genNode:    genNode{start: 43, end: 60},
			identifier: "s",
			typ:        "String",
		},
	}
	resultPrivateDecl := result.Tasks[0].PrvtDecls
	cmpOptions := append(commonCmpopts, cmp.AllowUnexported(decl{}))
	if diff := cmp.Diff(
		expectedPrivateDecl, resultPrivateDecl, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected task private declaration:\n%s", diff)
	}
}

func TestTaskCommand(t *testing.T) {
	inputPath := "testdata/task_command.wdl"
	result, err := Antlr4Parse(inputPath)
	expectedCommand := []string{
		"\n        echo \"Hello world\"\n    ",
	}
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultCommand := result.Tasks[0].Command
	if diff := cmp.Diff(expectedCommand, resultCommand); diff != "" {
		t.Errorf("unexpected task command:\n%s", diff)
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

	expectedOutput := []*decl{
		{
			genNode:    genNode{start: 47, end: 73},
			identifier: "output_file",
			typ:        "File",
		},
	}
	resultOutput := result.Tasks[0].Outputs
	cmpOptions := append(commonCmpopts, cmp.AllowUnexported(decl{}))
	if diff := cmp.Diff(
		expectedOutput, resultOutput, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected task output:\n%s", diff)
	}
}

func TestTaskRuntime(t *testing.T) {
	inputPath := "testdata/task_runtime.wdl"
	expectedRuntime := map[identifier]*exprRPN{
		newIdentifier("container", false): {},
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultRuntime := result.Tasks[0].Runtime
	if diff := cmp.Diff(expectedRuntime, resultRuntime); diff != "" {
		t.Errorf("unexpected task runtime:\n%s", diff)
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
	resultMeta := result.Tasks[0].Meta
	if diff := cmp.Diff(expectedMeta, resultMeta); diff != "" {
		t.Errorf("unexpected task metadata:\n%s", diff)
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
	resultParameterMeta := result.Tasks[0].ParameterMeta
	if diff := cmp.Diff(
		expectedParameterMeta, resultParameterMeta,
	); diff != "" {
		t.Errorf("unexpected task parameter metadata:\n%s", diff)
	}
}
