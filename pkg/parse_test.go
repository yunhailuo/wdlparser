package wdlparser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var commonCmpopts = cmp.Options{
	cmp.AllowUnexported(
		genNode{},
		identifier{},
		namedNode{},
		importSpec{},
		valueSpec{},
		Call{},
		expression{},
		value{},
	),
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

	expectedImports := []importSpec{
		{
			namedNode: namedNode{
				genNode: genNode{start: 13, end: 29},
				name:    newIdentifier("test", false),
			},
			uri: &exprRPN{
				value{String, "test.wdl"},
			},
			importAliases: map[string]string{},
		},
		{
			namedNode: namedNode{
				genNode: genNode{start: 31, end: 88},
				name:    newIdentifier("analysis_tasks", false),
				alias:   "analysis",
			},
			uri: &exprRPN{
				value{String, "http://example.com/lib/analysis_tasks"},
			},
			importAliases: map[string]string{},
		},
		{
			namedNode: namedNode{
				genNode: genNode{start: 90, end: 216},
				name:    newIdentifier("stdlib", false),
			},
			uri: &exprRPN{
				value{String, "https://example.com/lib/stdlib.wdl"},
			},
			importAliases: map[string]string{
				"Parent":     "Parent2",
				"Child":      "Child2",
				"GrandChild": "GrandChild2",
			},
		},
	}
	resultImports := []importSpec{}
	for i := range result.Imports {
		resultImports = append(resultImports, *result.Imports[i])
	}
	if diff := cmp.Diff(
		expectedImports, resultImports, commonCmpopts...,
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

	expectedInput := []*valueSpec{
		newValueSpec(50, 65, "input_str", "String"),
		newValueSpec(75, 94, "input_file_path", "File"),
	}
	resultInput := result.Workflow.Inputs
	if diff := cmp.Diff(
		expectedInput, resultInput, commonCmpopts...,
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

	expectedPrivateDecl := []*valueSpec{
		{
			genNode: genNode{start: 47, end: 64},
			name:    newIdentifier("s", false),
			typ:     "String",
			value:   &exprRPN{value{String, "Hello"}},
		},
	}
	resultPrivateDecl := result.Workflow.PrvtDecls
	if diff := cmp.Diff(
		expectedPrivateDecl, resultPrivateDecl, commonCmpopts...,
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
				genNode: genNode{start: 39, end: 168},
				name:    newIdentifier("Greeting", false),
				alias:   "hello",
			},
			Inputs: []*valueSpec{
				{
					genNode: genNode{start: 91, end: 113},
					name:    newIdentifier("first_name", true),
					typ:     "",
					value:   &exprRPN{newIdentifier("first_name", true)},
				},
				{
					genNode: genNode{start: 128, end: 144},
					name:    newIdentifier("last_name", true),
					typ:     "",
					value:   &exprRPN{value{String, "Luo"}},
				},
				{
					genNode: genNode{start: 159, end: 161},
					name:    newIdentifier("msg", true),
					typ:     "",
					value:   &exprRPN{newIdentifier("msg", true)},
				},
			},
		},
		{
			namedNode: namedNode{
				genNode: genNode{start: 174, end: 231},
				name:    newIdentifier("Goodbye", false),
			},
			After: "hello",
			Inputs: []*valueSpec{
				{
					genNode: genNode{start: 208, end: 228},
					name:    newIdentifier("first_name", true),
					typ:     "",
					value:   &exprRPN{value{String, "Yunhai"}},
				},
			},
		},
	}
	resultCalls := result.Workflow.Calls
	if diff := cmp.Diff(
		expectCalls, resultCalls, commonCmpopts...,
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

	expectedOutput := []*valueSpec{
		{
			genNode: genNode{start: 52, end: 87},
			name:    newIdentifier("output_file", false),
			typ:     "File",
			value:   &exprRPN{value{String, "/Path/to/output"}},
		},
	}
	resultOutput := result.Workflow.Outputs
	if diff := cmp.Diff(
		expectedOutput, resultOutput, commonCmpopts...,
	); diff != "" {
		t.Errorf("unexpected workflow output:\n%s", diff)
	}
}

func TestWorkflowMeta(t *testing.T) {
	inputPath := "testdata/workflow_meta.wdl"
	expectedMeta := []*valueSpec{
		{
			genNode: genNode{start: 48, end: 67},
			name:    newIdentifier("author", false),
			typ:     "",
			value:   &exprRPN{`"Yunhai Luo"`},
		},
		{
			genNode: genNode{start: 77, end: 88},
			name:    newIdentifier("version", false),
			typ:     "",
			value:   &exprRPN{"1.1"},
		},
		{
			genNode: genNode{start: 98, end: 112},
			name:    newIdentifier("for", false),
			typ:     "",
			value:   &exprRPN{`"workflow"`},
		},
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultMeta := result.Workflow.Meta
	if diff := cmp.Diff(
		expectedMeta, resultMeta, commonCmpopts...,
	); diff != "" {
		t.Errorf("unexpected workflow metadata:\n%s", diff)
	}
}

func TestWorkflowParameterMeta(t *testing.T) {
	inputPath := "testdata/workflow_parameter_meta.wdl"
	expectedParameterMeta := []*valueSpec{
		{
			genNode: genNode{start: 67, end: 129},
			name:    newIdentifier("name", false),
			typ:     "",
			value:   &exprRPN{`{help:"A name for workflow input"}`},
		},
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultParameterMeta := result.Workflow.ParameterMeta
	if diff := cmp.Diff(
		expectedParameterMeta, resultParameterMeta, commonCmpopts...,
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

	expectedInput := []*valueSpec{
		{
			genNode: genNode{start: 46, end: 66},
			name:    newIdentifier("name", false),
			typ:     "String",
			value:   &exprRPN{value{String, "World"}},
		},
		{
			genNode: genNode{start: 76, end: 95},
			name:    newIdentifier("input_file_path", false),
			typ:     "File",
			value:   &exprRPN{},
		},
	}
	resultInput := result.Tasks[0].Inputs
	if diff := cmp.Diff(
		expectedInput, resultInput, commonCmpopts...,
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

	expectedPrivateDecl := []*valueSpec{
		{
			genNode: genNode{start: 43, end: 60},
			name:    newIdentifier("s", false),
			typ:     "String",
			value:   &exprRPN{value{String, "Hello"}},
		},
	}
	resultPrivateDecl := result.Tasks[0].PrvtDecls
	if diff := cmp.Diff(
		expectedPrivateDecl, resultPrivateDecl, commonCmpopts...,
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

	expectedOutput := []*valueSpec{
		{
			genNode: genNode{start: 47, end: 73},
			name:    newIdentifier("output_file", false),
			typ:     "File",
			value:   &newExpression(0, 0).rpn,
		},
	}
	resultOutput := result.Tasks[0].Outputs
	if diff := cmp.Diff(
		expectedOutput, resultOutput, commonCmpopts...,
	); diff != "" {
		t.Errorf("unexpected task output:\n%s", diff)
	}
}

func TestTaskRuntime(t *testing.T) {
	inputPath := "testdata/task_runtime.wdl"
	expectedRuntime := []*valueSpec{
		{
			genNode: genNode{start: 50, end: 75},
			name:    newIdentifier("container", false),
			typ:     "",
			value:   &exprRPN{value{String, "ubuntu:latest"}},
		},
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultRuntime := result.Tasks[0].Runtime
	if diff := cmp.Diff(
		expectedRuntime, resultRuntime, commonCmpopts...,
	); diff != "" {
		t.Errorf("unexpected task runtime:\n%s", diff)
	}
}

func TestTaskMeta(t *testing.T) {
	inputPath := "testdata/task_meta.wdl"
	expectedMeta := []*valueSpec{
		{
			genNode: genNode{start: 44, end: 63},
			name:    newIdentifier("author", false),
			typ:     "",
			value:   &exprRPN{`"Yunhai Luo"`},
		},
		{
			genNode: genNode{start: 73, end: 84},
			name:    newIdentifier("version", false),
			typ:     "",
			value:   &exprRPN{"1.1"},
		},
		{
			genNode: genNode{start: 94, end: 104},
			name:    newIdentifier("for", false),
			typ:     "",
			value:   &exprRPN{`"task"`},
		},
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultMeta := result.Tasks[0].Meta
	if diff := cmp.Diff(
		expectedMeta, resultMeta, commonCmpopts...,
	); diff != "" {
		t.Errorf("unexpected task metadata:\n%s", diff)
	}
}

func TestTaskParameterMeta(t *testing.T) {
	inputPath := "testdata/task_parameter_meta.wdl"
	expectedParameterMeta := []*valueSpec{
		{
			genNode: genNode{start: 63, end: 122},
			name:    newIdentifier("name", false),
			typ:     "",
			value:   &exprRPN{`{help:"One name as task input"}`},
		},
	}
	result, err := Antlr4Parse(inputPath)
	if err != nil {
		t.Errorf(
			"Found %d errors in %q, expect no errors", len(err), inputPath,
		)
	}
	resultParameterMeta := result.Tasks[0].ParameterMeta
	if diff := cmp.Diff(
		expectedParameterMeta, resultParameterMeta, commonCmpopts...,
	); diff != "" {
		t.Errorf("unexpected task parameter metadata:\n%s", diff)
	}
}
