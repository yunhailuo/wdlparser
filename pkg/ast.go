package wdlparser

import (
	"fmt"
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
	mtd                 // metadata
	pmt                 // parameter metadata
	dcl                 // general declaration
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
	addChild(node) error
}

// A genNode is a concrete type of the node interface.
type genNode struct {
	start, end int
	parent     node
	children   []node
}

func (v *genNode) getStart() int         { return v.start }
func (v *genNode) getEnd() int           { return v.end }
func (*genNode) getKind() nodeKind       { return par }
func (v *genNode) getParent() node       { return v.parent }
func (v *genNode) setParent(parent node) { v.parent = parent }
func (v *genNode) getChildren() []node   { return v.children }

func (p *genNode) addChild(n node) error {
	newStart := n.getStart()
	newEnd := n.getEnd()
	for _, child := range p.children {
		if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
			return fmt.Errorf(
				"failed to add child; an existing child with"+
					" identical start and end has been found: %v", child,
			)
		}
	}
	p.children = append(p.children, n)
	return nil
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

// Declarations
// A decl represents a declaration.
type (
	declType string
	decl     struct {
		genNode
		identifier     string
		initialization exprRPN
		typ            declType
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

func (p *WDL) addChild(n node) error {
	switch c := n.(type) {
	case *importSpec:
		p.Imports = append(p.Imports, c)
	case *Workflow:
		if p.Workflow != nil {
			return fmt.Errorf(
				"a workflow is already defined in this WDL: %v;"+
					" cannot take another %T child",
				p.Workflow,
				n,
			)
		}
		p.Workflow = c
	case *Task:
		p.Tasks = append(p.Tasks, c)
	case *decl:
		p.Structs = append(p.Structs, c)
	default:
		return fmt.Errorf("WDL cannot have direct %T child: %v", n, n)
	}
	return nil
}

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
	Inputs []*keyValue
}

func NewCall(start, end int, name string) *Call {
	call := new(Call)
	call.namedNode = *newNamedNode(start, end, name)
	return call
}

func (*Call) getKind() nodeKind { return cal }

func (p *Call) addChild(n node) error {
	switch c := n.(type) {
	case *keyValue:
		p.Inputs = append(p.Inputs, c)
	default:
		// TODO: remove this support on child with arbitrary kind
		newStart := n.getStart()
		newEnd := n.getEnd()
		for _, child := range p.children {
			if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
				return fmt.Errorf(
					"failed to add child; an existing child with"+
						" identical start and end has been found: %v", child,
				)
			}
		}
		p.children = append(p.children, c)
	}
	return nil
}

// A Task represents one parsed task.
type Task struct {
	namedNode
	Inputs        []*decl
	PrvtDecls     []*decl
	Outputs       []*decl
	Command       []string
	Runtime       map[string]string
	Meta          map[string]string
	ParameterMeta map[string]string
}

func NewTask(start, end int, name string) *Task {
	task := new(Task)
	task.namedNode = *newNamedNode(start, end, name)
	task.Runtime = make(map[string]string)
	task.Meta = make(map[string]string)
	task.ParameterMeta = make(map[string]string)
	return task
}

func (*Task) getKind() nodeKind { return tsk }
