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
		for k, i := range workflow.GetInput() {
			wfRaw.inputs[k] = string(i.GetValue())
		}
		for k, o := range workflow.GetOutput() {
			wfRaw.outputs[k] = string(o.GetValue())
		}
		for k, m := range workflow.GetMetadata() {
			wfRaw.meta[k] = string(m.GetValue())
		}
		for k, p := range workflow.GetParameterMetadata() {
			wfRaw.parameterMeta[k] = string(p.GetValue())
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
		command                      []string
		inputs, privateDecl, outputs map[string]string
		runtime, meta, parameterMeta map[string]string
	}
	expectedTaskRaw := map[string]taskRaw{
		"WriteGreeting": {
			inputs:      map[string]string{"name": ""},
			privateDecl: map[string]string{"s": `"Hello"`},
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
		tRaw := taskRaw{
			inputs:        make(map[string]string),
			privateDecl:   make(map[string]string),
			command:       task.Command,
			outputs:       make(map[string]string),
			runtime:       make(map[string]string),
			meta:          make(map[string]string),
			parameterMeta: make(map[string]string),
		}
		for k, i := range task.GetInput() {
			tRaw.inputs[k] = string(i.GetValue())
		}
		for k, prv := range task.GetPrivateDecl() {
			tRaw.privateDecl[k] = string(prv.GetValue())
		}
		for k, o := range task.GetOutput() {
			tRaw.outputs[k] = string(o.GetValue())
		}
		for k, r := range task.GetRuntime() {
			tRaw.runtime[k] = string(r.GetValue())
		}
		for k, m := range task.GetMetadata() {
			tRaw.meta[k] = string(m.GetValue())
		}
		for k, p := range task.GetParameterMetadata() {
			tRaw.parameterMeta[k] = string(p.GetValue())
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
