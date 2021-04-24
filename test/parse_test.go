package wdlparser

import (
	"reflect"
	"testing"

	wdlparser "github.com/yunhailuo/wdlparser/pkg"
)

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
	expectedImportPaths := map[string]string{
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

func TestWorkflow(t *testing.T) {
	inputPath := "testdata/hello.wdl"
	type workflowRaw struct {
		inputs, outputs, meta, parameterMeta map[string]string
		elementCount                         int
	}
	expectedWorkflowRaw := map[string]workflowRaw{
		"HelloWorld": {
			elementCount: 5,
			inputs:       map[string]string{"wf_input_name": ""},
			outputs: map[string]string{
				"wf_output_greeting": "WriteGreeting.output_greeting",
			},
			meta: map[string]string{
				"author": `"Yunhai Luo"`, "for": `"workflow"`, "version": "1.1",
			},
			parameterMeta: map[string]string{
				"name": `{help:"A name for workflow input"}`},
		},
	}

	result, _ := wdlparser.Antlr4Parse(inputPath)
	resultWorkflowRaw := make(map[string]workflowRaw)
	for name, workflow := range result.GetWorkflow() {
		wfRaw := workflowRaw{
			elementCount:  len(workflow.Elements),
			inputs:        make(map[string]string),
			outputs:       make(map[string]string),
			meta:          make(map[string]string),
			parameterMeta: make(map[string]string),
		}
		for k, sym := range workflow.Inputs {
			wfRaw.inputs[k] = sym.GetRaw()
		}
		for k, sym := range workflow.Outputs {
			wfRaw.outputs[k] = sym.GetRaw()
		}
		for k, sym := range workflow.Meta {
			wfRaw.meta[k] = sym.GetRaw()
		}
		for k, sym := range workflow.ParameterMeta {
			wfRaw.parameterMeta[k] = sym.GetRaw()
		}
		resultWorkflowRaw[name] = wfRaw
	}
	if !reflect.DeepEqual(resultWorkflowRaw, expectedWorkflowRaw) {
		t.Errorf(
			"Found workflow %v, expect workflow %v",
			resultWorkflowRaw, expectedWorkflowRaw,
		)
	}
}

func TestTask(t *testing.T) {
	inputPath := "testdata/hello.wdl"
	type taskRaw struct {
		command                                       []string
		inputs, outputs, runtime, meta, parameterMeta map[string]string
	}
	expectedPrivateDecl := map[string]string{"s": `"Hello"`}
	expectedTaskRaw := map[string]taskRaw{
		"WriteGreeting": {
			inputs: map[string]string{"name": ""},
			command: []string{
				"\n        echo ", "~{s}", `" "`, "~{name}", "\n    ",
			},
			outputs: map[string]string{
				"output_greeting": "stdout()",
			},
			runtime: map[string]string{"container": `"ubuntu:latest"`},
			meta: map[string]string{
				"author": `"Yunhai Luo"`, "for": `"task"`, "version": "1.1",
			},
			parameterMeta: map[string]string{
				"name": `{help:"One name as task input"}`},
		},
	}

	result, _ := wdlparser.Antlr4Parse(inputPath)
	resultTaskRaw := make(map[string]taskRaw)
	for name, task := range result.GetTask() {
		for k, v := range expectedPrivateDecl {
			sym, err := task.Resolve(k)
			if err != nil {
				t.Errorf("Failed to find private declaration %q", k)
			} else if sym.GetRaw() != v {
				t.Errorf(
					"Found private declaration %q being %q, expect %q",
					k, sym.GetRaw(), v,
				)
			}
		}
		tRaw := taskRaw{
			inputs:        make(map[string]string),
			command:       task.Command,
			outputs:       make(map[string]string),
			runtime:       make(map[string]string),
			meta:          make(map[string]string),
			parameterMeta: make(map[string]string),
		}
		for k, sym := range task.Inputs {
			tRaw.inputs[k] = sym.GetRaw()
		}
		for k, sym := range task.Outputs {
			tRaw.outputs[k] = sym.GetRaw()
		}
		for k, sym := range task.Runtime {
			tRaw.runtime[k] = sym.GetRaw()
		}
		for k, sym := range task.Meta {
			tRaw.meta[k] = sym.GetRaw()
		}
		for k, sym := range task.ParameterMeta {
			tRaw.parameterMeta[k] = sym.GetRaw()
		}
		resultTaskRaw[name] = tRaw
	}
	if !reflect.DeepEqual(resultTaskRaw, expectedTaskRaw) {
		t.Errorf(
			"Found task %v, expect task %v",
			resultTaskRaw, expectedTaskRaw,
		)
	}
}
