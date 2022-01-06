package wdlparser

import (
	"path"
	"strings"
)

type node interface {
	getStart() int // position of first character belonging to the node, 0-based
	getEnd() int   // position of last character belonging to the node, 0-based

	getParent() node
	setParent(node)
}

// A genNode is a concrete type of the node interface.
type genNode struct {
	start, end int
	parent     node
}

func (v *genNode) getStart() int         { return v.start }
func (v *genNode) getEnd() int           { return v.end }
func (v *genNode) getParent() node       { return v.parent }
func (v *genNode) setParent(parent node) { v.parent = parent }

type identifier struct {
	initialName string
	isReference bool // otherwise, this is a definition
}

func newIdentifier(initialName string, isReference bool) *identifier {
	return &identifier{
		initialName: initialName,
		isReference: isReference,
	}
}

// An namedNode represents a named language entity such as input, private
// declaration, output, runtime metadata or parameter metadata.
type namedNode struct {
	genNode // Implement node interface
	name    *identifier
	alias   string
}

func newNamedNode(start, end int, name string) *namedNode {
	return &namedNode{
		genNode{start: start, end: end},
		newIdentifier(name, false),
		"",
	}
}

// A valueSpec represents a declaration or a key/value
type valueSpec struct {
	genNode
	name  *identifier
	typ   string
	value *exprRPN
}

func newValueSpec(start, end int, identifier, rawType string) *valueSpec {
	d := new(valueSpec)
	d.genNode = genNode{start: start, end: end}
	d.name = newIdentifier(identifier, false)
	d.typ = rawType
	v := make(exprRPN, 0)
	d.value = &v
	return d
}

// A WDL represents a parsed WDL document.
type WDL struct {
	namedNode
	Path     string
	Version  string
	Imports  []*importSpec
	Workflow *Workflow
	Tasks    []*Task
	Structs  []*valueSpec
}

func NewWDL(wdlPath string, size int) *WDL {
	wdl := new(WDL)
	wdl.Path = wdlPath
	wdl.namedNode = *newNamedNode(
		0,
		size-1,
		strings.TrimSuffix(path.Base(wdlPath), ".wdl"),
	)
	return wdl
}

type importSpec struct {
	namedNode
	uri           *exprRPN
	importAliases map[string]string // key is original name and value is alias
}

func newImportSpec(start, end int, uri string) *importSpec {
	is := new(importSpec)
	v := make(exprRPN, 0)
	is.uri = &v
	is.namedNode = *newNamedNode(
		start, end, strings.TrimSuffix(path.Base(uri), ".wdl"),
	)
	is.importAliases = map[string]string{}
	return is
}

// A Workflow represents one parsed workflow.
type Workflow struct {
	namedNode
	Inputs        []*valueSpec
	PrvtDecls     []*valueSpec
	Outputs       []*valueSpec
	Calls         []*Call
	Meta          []*valueSpec
	ParameterMeta []*valueSpec
}

func NewWorkflow(start, end int, name string) *Workflow {
	workflow := new(Workflow)
	workflow.namedNode = *newNamedNode(start, end, name)
	return workflow
}

// A Call represents one parsed call in a workflow.
type Call struct {
	namedNode
	After  string
	Inputs []*valueSpec
}

func NewCall(start, end int, name string) *Call {
	call := new(Call)
	call.namedNode = *newNamedNode(start, end, name)
	return call
}

// A Task represents one parsed task.
type Task struct {
	namedNode
	Inputs        []*valueSpec
	PrvtDecls     []*valueSpec
	Outputs       []*valueSpec
	Command       []string
	Runtime       []*valueSpec
	Meta          []*valueSpec
	ParameterMeta []*valueSpec
}

func NewTask(start, end int, name string) *Task {
	task := new(Task)
	task.namedNode = *newNamedNode(start, end, name)
	return task
}
