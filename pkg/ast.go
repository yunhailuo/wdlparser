package wdlparser

import (
	"log"
	"path"
	"strings"
)

// nodeKind describes what WDL entity a node represents.
type nodeKind int

const (
	par nodeKind = iota // for parsing
	doc                 // WDL document
	imp                 // import
	wfl                 // workflow
	cal                 // call
	tsk                 // task
	ipt                 // input
	opt                 // output
	rnt                 // task runtime
	mtd                 // metadata
	pmt                 // parameter metadata
	dcl                 // general declaration
	exp                 // expression
)

type node interface {
	getStart() int // position of first character belonging to the node, 0-based
	getEnd() int   // position of last character belonging to the node, 0-based
	getKind() nodeKind

	// Ideally, children should be a list of unique nodes (i.e. a set).
	// Two nodes are considered identical if and only if the have the same
	// start and end.
	getParent() node
	setParent(node)
	getChildren() []node
	addChild(node)
}

// A genNode is a concrete type of the node interface.
type genNode struct {
	start, end int
	parent     node
	children   []node
}

func newGenNode(start, end int) *genNode {
	return &genNode{start: start, end: end}
}

func (v *genNode) getStart() int         { return v.start }
func (v *genNode) getEnd() int           { return v.end }
func (*genNode) getKind() nodeKind       { return par }
func (v *genNode) getParent() node       { return v.parent }
func (v *genNode) setParent(parent node) { v.parent = parent }
func (v *genNode) getChildren() []node   { return v.children }

func (v *genNode) addChild(n node) {
	newStart := n.getStart()
	newEnd := n.getEnd()
	for _, child := range v.children {
		if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
			return
		}
	}
	v.children = append(v.children, n)
}

// An namedNode represents a named language entity such as input, private
// declaration, output, runtime metadata or parameter metadata.
type namedNode struct {
	genNode     // Implement node interface
	name, alias string
}

func newNamedNode(start, end int, name string) *namedNode {
	return &namedNode{genNode{start: start, end: end}, name, ""}
}

// An decl represents a declaration.
type (
	declType  string
	declValue string
	decl      struct {
		genNode
		identifier     string
		initialization *expr
		typ            declType
		value          declValue
	}
)

func newDecl(start, end int, identifier, rawType, rawValue string) *decl {
	d := new(decl)
	d.genNode = genNode{start: start, end: end}
	d.identifier = identifier
	d.typ = declType(rawType)
	d.value = declValue(rawValue)
	return d
}

func (*decl) getKind() nodeKind { return dcl }

// A keyValue represents a key/value pair defined in call input, runtime,
// metadata or parameter metadata sections.
type keyValue struct {
	genNode
	key, value string
}

func newKeyValue(start, end int, key, value string) *keyValue {
	return &keyValue{genNode{start: start, end: end}, key, value}
}

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
	Inputs        *inputDecls
	PrvtDecls     []*decl
	Outputs       *outputDecls
	Calls         []*Call
	Meta          *metaSpec
	ParameterMeta *parameterMetaSpec
	Elements      []string
}

func NewWorkflow(start, end int, name string) *Workflow {
	workflow := new(Workflow)
	workflow.namedNode = *newNamedNode(start, end, name)
	return workflow
}

func (*Workflow) getKind() nodeKind { return wfl }

type inputDecls struct {
	genNode
	decls []*decl
}

func newInputDecls(start, end int) *inputDecls {
	id := new(inputDecls)
	id.genNode = genNode{start, end, nil, []node{}}
	return id
}

func (*inputDecls) getKind() nodeKind { return ipt }

func (v *inputDecls) getChildren() []node {
	ns := []node{}
	for d := range v.decls {
		ns = append(ns, v.decls[d])
	}
	return ns
}

func (v *inputDecls) addChild(n node) {
	d, isDecl := n.(*decl)
	if !isDecl {
		log.Fatalf("inputSpec can only have decl child, got %T", n)
	}
	newStart := n.getStart()
	newEnd := n.getEnd()
	for _, child := range v.children {
		if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
			return
		}
	}
	v.decls = append(v.decls, d)
}

// A Call represents one parsed call.
type Call struct {
	namedNode
	After  string
	Inputs []*keyValue
}

func NewCall(start, end int, name string) *Call {
	call := new(Call)
	call.namedNode = *newNamedNode(start, end, name)
	return call
}

func (*Call) getKind() nodeKind { return cal }

type outputDecls struct {
	genNode
	decls []*decl
}

func newOutputDecls(start, end int) *outputDecls {
	od := new(outputDecls)
	od.genNode = genNode{start, end, nil, []node{}}
	return od
}

func (*outputDecls) getKind() nodeKind { return opt }

func (v *outputDecls) getChildren() []node {
	ns := []node{}
	for d := range v.decls {
		ns = append(ns, v.decls[d])
	}
	return ns
}

func (v *outputDecls) addChild(n node) {
	d, isDecl := n.(*decl)
	if !isDecl {
		log.Fatalf("outputDecls can only have decl child, got %T", n)
	}
	newStart := n.getStart()
	newEnd := n.getEnd()
	for _, child := range v.children {
		if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
			return
		}
	}
	v.decls = append(v.decls, d)
}

type metaSpec struct {
	genNode
	keyValues map[string]string
}

func newMetaSpecs(start, end int) *metaSpec {
	ms := new(metaSpec)
	ms.genNode = genNode{start, end, nil, []node{}}
	ms.keyValues = map[string]string{}
	return ms
}

func (*metaSpec) getKind() nodeKind { return mtd }

type parameterMetaSpec struct {
	genNode
	keyValues map[string]string
}

func newParameterMetaSpecs(start, end int) *parameterMetaSpec {
	pms := new(parameterMetaSpec)
	pms.genNode = genNode{start, end, nil, []node{}}
	pms.keyValues = map[string]string{}
	return pms
}

func (*parameterMetaSpec) getKind() nodeKind { return pmt }

// A Task represents one parsed task.
type Task struct {
	namedNode
	Inputs        *inputDecls
	PrvtDecls     []*decl
	Outputs       *outputDecls
	Command       []string
	Runtime       *runtimeSpec
	Meta          *metaSpec
	ParameterMeta *parameterMetaSpec
}

func NewTask(start, end int, name string) *Task {
	task := new(Task)
	task.namedNode = *newNamedNode(start, end, name)
	return task
}

func (*Task) getKind() nodeKind { return tsk }

type runtimeSpec struct {
	genNode
	keyValues map[string]string
}

func newRuntimeSpecs(start, end int) *runtimeSpec {
	rs := new(runtimeSpec)
	rs.genNode = genNode{start, end, nil, []node{}}
	rs.keyValues = map[string]string{}
	return rs
}

func (*runtimeSpec) getKind() nodeKind { return rnt }
