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
	addChild(node) error
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
		initialization *expr
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

func (p *decl) addChild(n node) error {
	switch c := n.(type) {
	case *expr:
		if p.initialization != nil {
			return fmt.Errorf(
				"a declaration takes only one expression as initialization"+
					" and already has one: %v; cannot take another %T child",
				p.initialization,
				n,
			)
		}
		p.initialization = c
	default:
		return fmt.Errorf("declaration cannot have direct %T child: %v", n, n)
	}
	return nil
}

// inputDecls is an array of declarations only for inputs
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

func (p *inputDecls) addChild(n node) error {
	switch c := n.(type) {
	case *decl:
		newStart := n.getStart()
		newEnd := n.getEnd()
		for _, child := range p.decls {
			if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
				return fmt.Errorf(
					"failed to add child; an existing child with"+
						" identical start and end has been found: %v", child,
				)
			}
		}
		p.decls = append(p.decls, c)
	default:
		return fmt.Errorf(
			"input declaration spec cannot have direct %T child: %v", n, n,
		)
	}
	return nil
}

// outputDecls is an array of declarations only for outputs
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

func (p *outputDecls) addChild(n node) error {
	switch c := n.(type) {
	case *decl:
		newStart := n.getStart()
		newEnd := n.getEnd()
		for _, child := range p.decls {
			if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
				return fmt.Errorf(
					"failed to add child; an existing child with"+
						" identical start and end has been found: %v", child,
				)
			}
		}
		p.decls = append(p.decls, c)
	default:
		return fmt.Errorf(
			"output declaration spec cannot have direct %T child: %v", n, n,
		)
	}
	return nil
}

// A keyValue represents a key/value pair defined in call input, runtime,
// metadata or parameter metadata sections.
type keyValue struct {
	genNode
	key, value string
}

func newKeyValue(start, end int, key, value string) *keyValue {
	return &keyValue{genNode{start: start, end: end}, key, value}
}

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

func (p *Workflow) addChild(n node) error {
	switch c := n.(type) {
	case *inputDecls:
		if p.Inputs != nil {
			return fmt.Errorf(
				"workflow inputs are already defined in one single set: %v;"+
					" cannot take another %T child",
				p.Inputs,
				n,
			)
		}
		p.Inputs = c
	case *decl:
		p.PrvtDecls = append(p.PrvtDecls, c)
	case *outputDecls:
		if p.Outputs != nil {
			return fmt.Errorf(
				"workflow outputs are already defined in one single set: %v;"+
					" cannot take another %T child",
				p.Outputs,
				n,
			)
		}
		p.Outputs = c
	case *Call:
		p.Calls = append(p.Calls, c)
	case *metaSpec:
		if p.Meta != nil {
			return fmt.Errorf(
				"workflow metadata are already defined in one single set: %v;"+
					" cannot take another %T child",
				p.Meta,
				n,
			)
		}
		p.Meta = c
	case *parameterMetaSpec:
		if p.ParameterMeta != nil {
			return fmt.Errorf(
				"metadata about workflow input/output parameter are already"+
					" defined in one single set: %v;"+
					" cannot take another %T child",
				p.ParameterMeta,
				n,
			)
		}
		p.ParameterMeta = c
	default:
		return fmt.Errorf("workflow cannot have direct %T child: %v", n, n)
	}
	return nil
}

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

func (p *Task) addChild(n node) error {
	switch c := n.(type) {
	case *inputDecls:
		if p.Inputs != nil {
			return fmt.Errorf(
				"task inputs are already defined in one single set: %v;"+
					" cannot take another %T child",
				p.Inputs,
				n,
			)
		}
		p.Inputs = c
	case *decl:
		p.PrvtDecls = append(p.PrvtDecls, c)
	case *outputDecls:
		if p.Outputs != nil {
			return fmt.Errorf(
				"task outputs are already defined in one single set: %v;"+
					" cannot take another %T child",
				p.Outputs,
				n,
			)
		}
		p.Outputs = c
	case *runtimeSpec:
		if p.Runtime != nil {
			return fmt.Errorf(
				"task runtime is already defined in one single set: %v;"+
					" cannot take another %T child",
				p.Meta,
				n,
			)
		}
		p.Runtime = c
	case *metaSpec:
		if p.Meta != nil {
			return fmt.Errorf(
				"task metadata are already defined in one single set: %v;"+
					" cannot take another %T child",
				p.Meta,
				n,
			)
		}
		p.Meta = c
	case *parameterMetaSpec:
		if p.ParameterMeta != nil {
			return fmt.Errorf(
				"metadata about task input/output parameter are already"+
					" defined in one single set: %v;"+
					" cannot take another %T child",
				p.ParameterMeta,
				n,
			)
		}
		p.ParameterMeta = c
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
