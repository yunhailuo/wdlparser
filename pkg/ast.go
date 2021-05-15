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

// An object represents a named language entity such as input, private
// declaration, output, runtime metadata or parameter metadata.
type object struct {
	// For node interface
	start, end int
	kind       nodeKind
	parent     node
	children   []node

	// Specific for object
	alias, name string
}

func newObject(
	start, end int, kind nodeKind, name string,
) *object {
	o := new(object)
	o.start = start
	o.end = end
	o.kind = kind
	o.name = name
	return o
}

func (o *object) getStart() int         { return o.start }
func (o *object) getEnd() int           { return o.end }
func (o *object) getKind() nodeKind     { return o.kind }
func (o *object) setKind(kind nodeKind) { o.kind = kind }

func (o *object) getParent() node { return o.parent }

func (o *object) setParent(parent node) {
	o.parent = parent
	parent.addChild(o)
}

func (o *object) getChildren() []node { return o.children }

func (o *object) addChild(n node) {
	newStart := n.getStart()
	newEnd := n.getEnd()
	for _, child := range o.children {
		if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
			return
		}
	}
	o.children = append(o.children, n)
	// Note that this add child method will not set parent on node `n`
}

func (o *object) setAlias(a string) { o.alias = a }
func (o *object) getName() string   { return o.name }
func (o *object) setName(n string)  { o.name = n }

type declType string

type declValue string

// An decl represents a declaration.
type decl struct {
	object
	expr  *expr
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
