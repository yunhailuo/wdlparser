package wdlparser

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var commonCmpopts = cmp.Options{
	cmp.AllowUnexported(genNode{}, namedNode{}),
	cmpopts.IgnoreFields(genNode{}, "parent", "children"),
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

	input1 := newDecl(50, 65, "input_str", "String")
	input2 := newDecl(75, 94, "input_file_path", "File")
	expectedInput := []*decl{input1, input2}
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
	cmpOptions := append(
		commonCmpopts,
		cmp.AllowUnexported(decl{}),
	)
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
				name:    "Greeting",
				alias:   "hello",
			},
			Inputs: []*keyValue{
				newKeyValue(91, 113, "first_name", "first_name"),
				newKeyValue(128, 144, "last_name", `"Luo"`),
			},
		},
		{
			namedNode: namedNode{
				genNode: genNode{start: 156, end: 213}, name: "Goodbye",
			},
			After: "hello",
			Inputs: []*keyValue{
				newKeyValue(190, 210, "first_name", `"Yunhai"`),
			},
		},
	}
	resultCalls := result.Workflow.Calls
	cmpOptions := append(commonCmpopts, cmp.AllowUnexported(Call{}, keyValue{}))
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
	cmpOptions := append(
		commonCmpopts,
		cmp.AllowUnexported(decl{}),
	)
	if diff := cmp.Diff(
		expectedOutput, resultOutput, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected workflow calls:\n%s", diff)
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
	cmpOptions := append(
		commonCmpopts,
		cmp.AllowUnexported(decl{}),
	)
	if diff := cmp.Diff(
		expectedInput, resultInput, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected workflow calls:\n%s", diff)
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
	cmpOptions := append(
		commonCmpopts,
		cmp.AllowUnexported(decl{}),
	)
	if diff := cmp.Diff(
		expectedPrivateDecl, resultPrivateDecl, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected workflow calls:\n%s", diff)
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

	expectedOutput := []*decl{
		{
			genNode:    genNode{start: 47, end: 73},
			identifier: "output_file",
			typ:        "File",
		},
	}
	resultOutput := result.Tasks[0].Outputs
	cmpOptions := append(
		commonCmpopts,
		cmp.AllowUnexported(decl{}),
	)
	if diff := cmp.Diff(
		expectedOutput, resultOutput, cmpOptions...,
	); diff != "" {
		t.Errorf("unexpected workflow calls:\n%s", diff)
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
	resultParameterMeta := result.Tasks[0].ParameterMeta
	if !reflect.DeepEqual(resultParameterMeta, expectedParameterMeta) {
		t.Errorf(
			"Found task parameter metadata %v, expect %v",
			resultParameterMeta, expectedParameterMeta,
		)
	}
}
