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
	getParent() node
	getKind() nodeKind
	setKind(nodeKind)
}

type identType string

type identValue string

// An Object represents a generic (private) declaration, input, output, runtime
// metadata or parameter metadata entry.
type object struct {
	start, end  int
	parent      node
	kind        nodeKind
	alias, name string
	typ         identType
	value       identValue
}

func newObject(
	start, end int, kind nodeKind, name, rawType, rawValue string,
) *object {
	s := new(object)
	s.start = start
	s.end = end
	s.kind = kind
	s.name = name
	s.typ = identType(rawType)
	s.value = identValue(rawValue)
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

// A WDL represents a parsed WDL document.
type WDL struct {
	object
	Path     string
	Version  string
	Imports  []*WDL
	Workflow *Workflow
	Tasks    []*Task
	Structs  []*object
}

func NewWDL(wdlPath string, size int) *WDL {
	wdl := new(WDL)
	wdl.Path = wdlPath
	wdl.object = *newObject(
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
	object
	Inputs, PrvtDecls, Outputs []*object
	Calls                      []*Call
	Meta, ParameterMeta        map[string]*object
	Elements                   []string
}

func NewWorkflow(start, end int, name string) *Workflow {
	workflow := new(Workflow)
	workflow.object = *newObject(start, end, wfl, name, "", "")
	workflow.Meta = make(map[string]*object)
	workflow.ParameterMeta = make(map[string]*object)
	return workflow
}

// A Call represents one parsed call
type Call struct {
	object
	After  string
	Inputs []*object
}

func NewCall(start, end int, name string) *Call {
	call := new(Call)
	call.object = *newObject(start, end, cal, name, "", "")
	return call
}

// A Task represents one parsed task
type Task struct {
	object
	Inputs, PrvtDecls, Outputs   []*object
	Command                      []string
	Runtime, Meta, ParameterMeta map[string]*object
}

func NewTask(start, end int, name string) *Task {
	task := new(Task)
	task.object = *newObject(start, end, tsk, name, "", "")
	task.Runtime = make(map[string]*object)
	task.Meta = make(map[string]*object)
	task.ParameterMeta = make(map[string]*object)
	return task
}
