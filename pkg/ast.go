package wdlparser

import (
	"path"
	"strings"
)

// nodeKind describes what WDL entity a node represents.
type nodeKind int

const (
	_   nodeKind = iota // leave 0 as nodeKind zero value; start from 1
	par                 // for parsing only
	doc                 // WDL document
	imp                 // import
	wfl                 // workflow
	cal                 // call
	tsk                 // task
	ipt                 // input
	opt                 // output
	mtd                 // metadata
	pmt                 // parameter metadata
	dcl                 // general declaration
)

type node interface {
	getStart() int // position of first character belonging to the node, 0-based
	getEnd() int   // position of last character belonging to the node, 0-based
	getKind() nodeKind

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
func (*genNode) getKind() nodeKind       { return par }
func (v *genNode) getParent() node       { return v.parent }
func (v *genNode) setParent(parent node) { v.parent = parent }

type identifier struct {
	initialName string
	isReference bool // otherwise, this is a definition
}

func newIdentifier(initialName string, isReference bool) identifier {
	return identifier{
		initialName: initialName,
		isReference: isReference,
	}
}

// An namedNode represents a named language entity such as input, private
// declaration, output, runtime metadata or parameter metadata.
type namedNode struct {
	genNode // Implement node interface
	name    identifier
	alias   string
}

func newNamedNode(start, end int, name string) *namedNode {
	return &namedNode{
		genNode{start: start, end: end},
		newIdentifier(name, false),
		"",
	}
}

// Declarations
// A decl represents a declaration.
type (
	declType string
	decl     struct {
		genNode
		identifier string
		value      exprRPN
		typ        declType
	}
)

func newDecl(start, end int, identifier, rawType string) *decl {
	d := new(decl)
	d.genNode = genNode{start: start, end: end}
	d.identifier = identifier
	d.typ = declType(rawType)
	return d
}

func (*decl) getKind() nodeKind { return dcl }

// A WDL represents a parsed WDL document.
type WDL struct {
	namedNode
	Path     string
	Version  string
	Imports  []*importSpec
	Workflow *Workflow
	Tasks    []*Task
	Structs  []*decl
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

func (*WDL) getKind() nodeKind { return doc }

type importSpec struct {
	namedNode
	uri           string
	importAliases map[string]string // key is original name and value is alias
}

func newImportSpec(start, end int, uri string) *importSpec {
	is := new(importSpec)
	is.uri = uri
	is.namedNode = *newNamedNode(
		start, end, strings.TrimSuffix(path.Base(uri), ".wdl"),
	)
	is.importAliases = map[string]string{}
	return is
}

func (*importSpec) getKind() nodeKind { return imp }

// A Workflow represents one parsed workflow.
type Workflow struct {
	namedNode
	Inputs        []*decl
	PrvtDecls     []*decl
	Outputs       []*decl
	Calls         []*Call
	Meta          map[string]string
	ParameterMeta map[string]string
	Elements      []string
}

func NewWorkflow(start, end int, name string) *Workflow {
	workflow := new(Workflow)
	workflow.namedNode = *newNamedNode(start, end, name)
	workflow.Meta = make(map[string]string)
	workflow.ParameterMeta = make(map[string]string)
	return workflow
}

func (*Workflow) getKind() nodeKind { return wfl }

// A Call represents one parsed call in a workflow.
type Call struct {
	namedNode
	After  string
	Inputs map[identifier]*exprRPN
}

func NewCall(start, end int, name string) *Call {
	call := new(Call)
	call.namedNode = *newNamedNode(start, end, name)
	call.Inputs = make(map[identifier]*exprRPN)
	return call
}

func (*Call) getKind() nodeKind { return cal }

// A Task represents one parsed task.
type Task struct {
	namedNode
	Inputs        []*decl
	PrvtDecls     []*decl
	Outputs       []*decl
	Command       []string
	Runtime       map[identifier]*exprRPN
	Meta          map[string]string
	ParameterMeta map[string]string
}

func NewTask(start, end int, name string) *Task {
	task := new(Task)
	task.namedNode = *newNamedNode(start, end, name)
	task.Runtime = make(map[identifier]*exprRPN)
	task.Meta = make(map[string]string)
	task.ParameterMeta = make(map[string]string)
	return task
}

func (*Task) getKind() nodeKind { return tsk }
