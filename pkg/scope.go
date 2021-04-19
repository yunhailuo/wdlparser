package wdlparser

import (
	"fmt"
	"path"
	"strings"
)

type Scoper interface {
	GetParent() Scoper
	SetParent(Scoper)
	GetChildren() []Scoper
	GetSymbolMap() map[string]Symboler
	Define(Symboler) error
	Resolve(string) (Symboler, error)
}

type Scope struct {
	parent    Scoper
	children  []Scoper
	symbolMap map[string]Symboler
}

func NewScope() *Scope {
	Scope := new(Scope)
	Scope.symbolMap = make(map[string]Symboler)
	return Scope
}

func (s *Scope) GetParent() Scoper {
	return s.parent
}

func (s *Scope) SetParent(parent Scoper) {
	s.parent = parent
}

func (s *Scope) GetChildren() []Scoper {
	return s.children
}

func (s *Scope) GetSymbolMap() map[string]Symboler {
	return s.symbolMap
}

func (s *Scope) Define(sym Symboler) error {
	if sym.GetName() == "" {
		return fmt.Errorf("Symbol %v doesn't have a valid name", sym)
	}
	if s.symbolMap == nil {
		return fmt.Errorf("SymbolMap of scope %v not initialized", s)
	}
	s.symbolMap[sym.GetName()] = sym
	return nil
}
func (s *Scope) Resolve(name string) (Symboler, error) {
	if sym, ok := s.symbolMap[name]; ok {
		return sym, nil
	}
	if s.GetParent() != nil {
		return s.GetParent().Resolve(name)
	}
	return new(Symbol), fmt.Errorf("%v not defined", name)
}

// WDL represnets a parsed WDL document.
// It is also the global scope of a parsing
type WDL struct {
	ScopedSymbol
	Path     string
	Version  string
	Imports  map[string]*Import
	Structs  []string
	Workflow *Workflow
	Tasks    map[string]*Task
}

func NewWDL(wdlPath string) *WDL {
	wdl := new(WDL)
	wdl.Path = wdlPath
	wdl.SetName(strings.TrimSuffix(path.Base(wdlPath), ".wdl"))
	wdl.SetType("document")
	wdl.Tasks = make(map[string]*Task)
	return wdl
}

// Import represents a parsed import
type Import struct {
	ScopedSymbol
	Wdl           *WDL
	Alias         string
	StructAliases map[string]string
}

func NewImport(wdlPath string) *Import {
	imp := new(Import)
	imp.Wdl = NewWDL(wdlPath)
	imp.Name = strings.TrimSuffix(path.Base(wdlPath), ".wdl")
	imp.Type = "import"
	imp.StructAliases = make(map[string]string)
	return imp
}

// Workflow records one parsed workflow
type Workflow struct {
	ScopedSymbol
	Elements []string
}

func NewWorkflow(name string) *Workflow {
	workflow := new(Workflow)
	workflow.SetName(name)
	workflow.SetType("workflow")
	workflow.symbolMap = make(map[string]Symboler)
	return workflow
}

// Workflow records one parsed workflow
type Task struct {
	ScopedSymbol
	Elements []string
}

func NewTask(name string) *Task {
	task := new(Task)
	task.SetName(name)
	task.SetType("task")
	return task
}
