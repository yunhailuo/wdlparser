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
	tsk                 // task
	Ipt                 // input
	Opt                 // output
	Rnt                 // runtime
	Mtd                 // metadata
	Pmt                 // parameter metadata
	Dcl                 // general declaration
)

type node interface {
	getStart() int // position of first character belonging to the node, 0-based
	getEnd() int   // position of last character belonging to the node, 0-based
	getParent() node
	getKind() nodeKind
	setKind(nodeKind)
}

type identType string

type identValue string

// An Object represents a generic (private) declaration, input, output, runtime
// metadata or parameter metadata entry.
type Object struct {
	start, end  int
	parent      node
	kind        nodeKind
	alias, name string
	typ         identType
	value       identValue
}

func NewObject(
	start, end int, kind nodeKind, name, rawType, rawValue string,
) *Object {
	s := new(Object)
	s.start = start
	s.end = end
	s.kind = kind
	s.name = name
	s.typ = identType(rawType)
	s.value = identValue(rawValue)
	return s
}

func (s *Object) getStart() int {
	return s.start
}

func (s *Object) getEnd() int {
	return s.end
}

func (s *Object) getParent() node {
	return s.parent
}

func (s *Object) setParent(parent node) {
	s.parent = parent
}

func (s *Object) getKind() nodeKind {
	return s.kind
}

func (s *Object) GetAlias() string {
	return s.alias
}

func (s *Object) setAlias(a string) {
	s.alias = a
}

func (s *Object) setKind(kind nodeKind) {
	s.kind = kind
}

func (s *Object) GetName() string {
	return s.name
}

// A WDL represents a parsed WDL document.
type WDL struct {
	Object
	Path     string
	Version  string
	Imports  []*WDL
	Workflow *Workflow
	Tasks    []*Task
	Structs  []*Object
}

func NewWDL(wdlPath string, size int) *WDL {
	wdl := new(WDL)
	wdl.Path = wdlPath
	wdl.Object = *NewObject(
		0,
		size-1,
		doc,
		strings.TrimSuffix(path.Base(wdlPath), ".wdl"),
		"",
		"",
	)
	return wdl
}

// A Workflow represents one parsed workflow
type Workflow struct {
	Object
	Inputs, PrvtDecls, Outputs []*Object
	Meta, ParameterMeta        map[string]*Object
	Elements                   []string
}

func NewWorkflow(start, end int, name string) *Workflow {
	workflow := new(Workflow)
	workflow.Object = *NewObject(start, end, wfl, name, "", "")
	workflow.Meta = make(map[string]*Object)
	workflow.ParameterMeta = make(map[string]*Object)
	return workflow
}

// A Task represents one parsed task
type Task struct {
	Object
	Inputs, PrvtDecls, Outputs   []*Object
	Command                      []string
	Runtime, Meta, ParameterMeta map[string]*Object
}

func NewTask(start, end int, name string) *Task {
	task := new(Task)
	task.Object = *NewObject(start, end, tsk, name, "", "")
	task.Runtime = make(map[string]*Object)
	task.Meta = make(map[string]*Object)
	task.ParameterMeta = make(map[string]*Object)
	return task
}
