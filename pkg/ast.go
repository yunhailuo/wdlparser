package wdlparser

import (
	"path"
	"strings"
)

// All named WDL entities implement the decl interface.
type decl interface {
	getAlias() string
	setAlias(string)
	getKind() objKind
	setKind(objKind)
	getName() string
	getType() identType
	GetValue() identValue
}

// ObjKind describes what WDL entity an object represents.
type objKind int

const (
	doc objKind = iota // WDL document
	imp                // import
	wfl                // workflow
	tsk                // task
	ipt                // input
	opt                // output
	rnt                // runtime
	mtd                // metadata
	pmt                // parameter metadata
	dcl                // general declaration
)

type identType string

type identValue string

// An object represents a generic (private) declaration, input, output, runtime
// metadata or parameter metadata entry.
type object struct {
	alias string
	kind  objKind
	name  string
	typ   identType
	value identValue
}

func newObject(kind objKind, name, rawType, rawValue string) *object {
	s := new(object)
	s.kind = kind
	s.name = name
	s.typ = identType(rawType)
	s.value = identValue(rawValue)
	return s
}

func (s *object) getAlias() string {
	return s.alias
}

func (s *object) setAlias(a string) {
	s.alias = a
}

func (s *object) getKind() objKind {
	return s.kind
}

func (s *object) setKind(kind objKind) {
	s.kind = kind
}

func (s *object) getName() string {
	return s.name
}

func (s *object) getType() identType {
	return s.typ
}

func (s *object) GetValue() identValue {
	return s.value
}

// All WDL namespaces implement the namespace interface.
type namespace interface {
	getParent() namespace
	setParent(namespace)
	getChildren() []namespace
	getDeclarations() []decl
	getDeclaration(objKind) map[string]decl
	addDeclaration(decl)
}

type scope struct {
	parent   namespace
	children []namespace
	body     []decl
}

func newScope() *scope {
	return new(scope)
}

func (s *scope) getParent() namespace {
	return s.parent
}

func (s *scope) setParent(parent namespace) {
	s.parent = parent
}

func (s *scope) getChildren() []namespace {
	return s.children
}

func (s *scope) getDeclarations() []decl {
	return s.body
}

func (s *scope) getDeclaration(k objKind) map[string]decl {
	ret := map[string]decl{}
	for _, d := range s.body {
		if d.getKind() == k {
			k := d.getName()
			if d.getAlias() != "" {
				k = d.getAlias()
			}
			ret[k] = d
		}
	}
	return ret
}

func (s *scope) addDeclaration(d decl) {
	s.body = append(s.body, d)
}

type scopedObject struct {
	scope
	object
}

func newScopedIdenifier(
	kind objKind, name, rawType, rawValue string,
) *scopedObject {
	si := new(scopedObject)
	si.scope = *newScope()
	si.object = *newObject(kind, name, rawType, rawValue)
	return si
}

// A WDL represents a parsed WDL document.
type WDL struct {
	scopedObject
	Path    string
	Version string
	Body    []object
}

func NewWDL(wdlPath string) *WDL {
	wdl := new(WDL)
	wdl.Path = wdlPath
	wdl.scopedObject = *newScopedIdenifier(
		doc,
		strings.TrimSuffix(path.Base(wdlPath), ".wdl"),
		"",
		"",
	)
	return wdl
}

func (wdl WDL) GetImports() map[string]*WDL {
	ret := map[string]*WDL{}
	for _, d := range wdl.getDeclarations() {
		if w, ok := d.(*WDL); ok {
			k := w.getName()
			if w.getAlias() != "" {
				k = w.getAlias()
			}
			ret[k] = w
		}
	}
	return ret
}

func (wdl WDL) GetWorkflow() map[string]*Workflow {
	ret := map[string]*Workflow{}
	for _, d := range wdl.getDeclarations() {
		if w, ok := d.(*Workflow); ok {
			ret[w.getName()] = w
		}
	}
	return ret
}

func (wdl WDL) GetTask() map[string]*Task {
	ret := map[string]*Task{}
	for _, d := range wdl.getDeclarations() {
		if w, ok := d.(*Task); ok {
			ret[w.getName()] = w
		}
	}
	return ret
}

// A Workflow represents one parsed workflow
type Workflow struct {
	scopedObject
	Elements []string
}

func NewWorkflow(name string) *Workflow {
	workflow := new(Workflow)
	workflow.scopedObject = *newScopedIdenifier(wfl, name, "", "")
	return workflow
}

func (wf Workflow) GetInput() map[string]decl {
	return wf.getDeclaration(ipt)
}

func (wf Workflow) GetOutput() map[string]decl {
	return wf.getDeclaration(opt)
}

func (wf Workflow) GetMetadata() map[string]decl {
	return wf.getDeclaration(mtd)
}

func (wf Workflow) GetParameterMetadata() map[string]decl {
	return wf.getDeclaration(pmt)
}

// A Task represents one parsed task
type Task struct {
	scopedObject
	Elements, Command []string
}

func NewTask(name string) *Task {
	task := new(Task)
	task.scopedObject = *newScopedIdenifier(tsk, name, "", "")
	return task
}

func (t Task) GetInput() map[string]decl {
	return t.getDeclaration(ipt)
}

func (t Task) GetPrivateDecl() map[string]decl {
	return t.getDeclaration(dcl)
}

func (t Task) GetOutput() map[string]decl {
	return t.getDeclaration(opt)
}

func (t Task) GetRuntime() map[string]decl {
	return t.getDeclaration(rnt)
}

func (t Task) GetMetadata() map[string]decl {
	return t.getDeclaration(mtd)
}

func (t Task) GetParameterMetadata() map[string]decl {
	return t.getDeclaration(pmt)
}
