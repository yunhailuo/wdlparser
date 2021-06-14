package wdlparser

import (
	"log"
	"path"
	"strings"
)

// nodeKind describes what WDL entity a node represents.
type nodeKind int

const (
	doc nodeKind = iota // WDL document
	imp                 // import
	wfl                 // workflow
	cal                 // call
	tsk                 // task
	ipt                 // input
	opt                 // output
	rnt                 // runtime
	mtd                 // metadata
	pmt                 // parameter metadata
	dcl                 // general declaration
	exp                 // expression
)

type node interface {
	getStart() int // position of first character belonging to the node, 0-based
	getEnd() int   // position of last character belonging to the node, 0-based
	getKind() nodeKind
	setKind(nodeKind)

	// Ideally, children should be a list of unique nodes (i.e. a set).
	// Two nodes are considered identical if and only if the have the same
	// start and end.
	getParent() node
	setParent(node)
	getChildren() []node
	addChild(node)
}

// A vertex is a concrete type of the node interface.
type vertex struct {
	start, end int
	kind       nodeKind
	parent     node
	children   []node
}

func (v *vertex) getStart() int         { return v.start }
func (v *vertex) getEnd() int           { return v.end }
func (v *vertex) getKind() nodeKind     { return v.kind }
func (v *vertex) setKind(kind nodeKind) { v.kind = kind }
func (v *vertex) getParent() node       { return v.parent }
func (v *vertex) setParent(parent node) { v.parent = parent }
func (v *vertex) getChildren() []node   { return v.children }

func (v *vertex) addChild(n node) {
	newStart := n.getStart()
	newEnd := n.getEnd()
	for _, child := range v.children {
		if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
			return
		}
	}
	v.children = append(v.children, n)
}

// An object represents a named language entity such as input, private
// declaration, output, runtime metadata or parameter metadata.
type object struct {
	vertex      // Implement node interface
	name, alias string
}

func newObject(
	start, end int, kind nodeKind, name string,
) *object {
	return &object{vertex{start: start, end: end, kind: kind}, name, ""}
}

// An decl represents a declaration.
type (
	declType  string
	declValue string
	decl      struct {
		vertex
		identifier string
		evaluator  evaluator
		typ        declType
		value      declValue
	}
)

func newDecl(
	start, end int, kind nodeKind, identifier, rawType, rawValue string,
) *decl {
	d := new(decl)
	d.vertex = vertex{start: start, end: end, kind: kind}
	d.identifier = identifier
	d.typ = declType(rawType)
	d.value = declValue(rawValue)
	return d
}

// A keyValue represents a key/value pair defined in call input, runtime,
// metadata or parameter metadata sections.
type keyValue struct {
	vertex
	key, value string
}

func newKeyValue(
	start, end int, kind nodeKind, key, value string,
) *keyValue {
	return &keyValue{vertex{start: start, end: end, kind: kind}, key, value}
}

// A WDL represents a parsed WDL document.
type WDL struct {
	object
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
	wdl.object = *newObject(
		0,
		size-1,
		doc,
		strings.TrimSuffix(path.Base(wdlPath), ".wdl"),
	)
	return wdl
}

type importSpec struct {
	object
	uri           string
	importAliases map[string]string // key is original name and value is alias
}

func newImportSpec(start, end int, uri string) *importSpec {
	is := new(importSpec)
	is.uri = uri
	is.object = *newObject(
		start, end, imp, strings.TrimSuffix(path.Base(uri), ".wdl"),
	)
	is.importAliases = map[string]string{}
	return is
}

func (v *importSpec) getKind() nodeKind { return imp }
func (v *importSpec) setKind(kind nodeKind) {
	log.Fatal("cannot setKind on importSpec node; it's imp node only")
}

// A Workflow represents one parsed workflow.
type Workflow struct {
	object
	Inputs, PrvtDecls, Outputs []*decl
	Calls                      []*Call
	Meta, ParameterMeta        map[string]*keyValue
	Elements                   []string
}

func NewWorkflow(start, end int, name string) *Workflow {
	workflow := new(Workflow)
	workflow.object = *newObject(start, end, wfl, name)
	workflow.Meta = make(map[string]*keyValue)
	workflow.ParameterMeta = make(map[string]*keyValue)
	return workflow
}

// A Call represents one parsed call.
type Call struct {
	object
	After  string
	Inputs []*keyValue
}

func NewCall(start, end int, name string) *Call {
	call := new(Call)
	call.object = *newObject(start, end, cal, name)
	return call
}

// A Task represents one parsed task.
type Task struct {
	object
	Inputs, PrvtDecls, Outputs   []*decl
	Command                      []string
	Runtime, Meta, ParameterMeta map[string]*keyValue
}

func NewTask(start, end int, name string) *Task {
	task := new(Task)
	task.object = *newObject(start, end, tsk, name)
	task.Runtime = make(map[string]*keyValue)
	task.Meta = make(map[string]*keyValue)
	task.ParameterMeta = make(map[string]*keyValue)
	return task
}
