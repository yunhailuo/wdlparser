package wdlparser

import (
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

// An object represents a named language entity such as input, private
// declaration, output, runtime metadata or parameter metadata.
type object struct {
	start, end  int
	parent      node
	children    []node
	kind        nodeKind
	alias, name string
}

func newObject(
	start, end int, kind nodeKind, name string,
) *object {
	s := new(object)
	s.start = start
	s.end = end
	s.kind = kind
	s.name = name
	return s
}

func (s *object) getStart() int {
	return s.start
}

func (s *object) getEnd() int {
	return s.end
}

func (s *object) getParent() node {
	return s.parent
}

func (s *object) setParent(parent node) {
	s.parent = parent
	parent.addChild(s)
}

func (s *object) getChildren() []node {
	return s.children
}

func (s *object) addChild(n node) {
	newStart := n.getStart()
	newEnd := n.getEnd()
	for _, child := range s.children {
		if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
			return
		}
	}
	s.children = append(s.children, n)
	// Note that this add child method will not set parent on node `n`
}

func (s *object) getKind() nodeKind {
	return s.kind
}

func (s *object) setKind(kind nodeKind) {
	s.kind = kind
}

func (s *object) setAlias(a string) {
	s.alias = a
}

func (s *object) getName() string {
	return s.name
}

func (s *object) setName(n string) {
	s.name = n
}

type declType string

type declValue string

// An decl represents a declaration.
type decl struct {
	object
	typ   declType
	value declValue
}

func newDecl(
	start, end int, kind nodeKind, name, rawType, rawValue string,
) *decl {
	d := new(decl)
	d.object = *newObject(start, end, kind, name)
	d.typ = declType(rawType)
	d.value = declValue(rawValue)
	return d
}

// A keyValue represents a key/value pair defined in call input, runtime,
// metadata or parameter metadata sections.
type keyValue struct {
	object
	value string
}

func newKeyValue(
	start, end int, kind nodeKind, key, value string,
) *keyValue {
	kv := new(keyValue)
	kv.object = *newObject(start, end, kind, key)
	kv.value = value
	return kv
}

// A WDL represents a parsed WDL document.
type WDL struct {
	object
	Path     string
	Version  string
	Imports  []*WDL
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

// A Workflow represents one parsed workflow
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

// A Call represents one parsed call
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

// A Task represents one parsed task
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
