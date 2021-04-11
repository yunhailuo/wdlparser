package wdlparser

import (
	"fmt"
	"path"
	"strings"
)

type Scope interface {
	GetParent() Scope
	GetChildren() []Scope
	GetSymbolMap() map[string]Symbol
	Define(Symbol) error
	Resolve(string) (Symbol, error)
}

type BaseScope struct {
	Parent    Scope
	Children  []Scope
	SymbolMap map[string]Symbol
}

func (s BaseScope) GetParent() Scope {
	return s.Parent
}

func (s BaseScope) GetChildren() []Scope {
	return s.Children
}

func (s BaseScope) GetSymbolMap() map[string]Symbol {
	return s.SymbolMap
}

func (s BaseScope) Define(sym Symbol) error {
	if sym.Name == "" {
		return fmt.Errorf("Symbol %v doesn't have a valid name", sym)
	}
	if s.SymbolMap == nil {
		return fmt.Errorf("SymbolMap of scope %v not initialized", s)
	}
	s.SymbolMap[sym.Name] = sym
	return nil
}
func (s BaseScope) Resolve(name string) (Symbol, error) {
	if sym, ok := s.SymbolMap[name]; ok {
		return sym, nil
	}
	if s.GetParent() != nil {
		return s.GetParent().Resolve(name)
	}
	return *new(Symbol), fmt.Errorf("%v not defined", name)
}

// ruleScope is designed to be temporary and used only for collecting
// information from subrules in ANTLR4 parser grammar
type ruleScope struct {
	BaseScope
}

func newRuleScope() *ruleScope {
	scp := new(ruleScope)
	scp.SymbolMap = make(map[string]Symbol)
	return scp
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
	wdl.Name = strings.TrimSuffix(path.Base(wdlPath), ".wdl")
	wdl.Type = "document"
	wdl.Imports = make(map[string]*Import)
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
	workflow.Name = name
	workflow.Type = "workflow"
	return workflow
}

// Workflow records one parsed workflow
type Task struct {
	ScopedSymbol
	Elements []string
}

func NewTask(name string) *Task {
	task := new(Task)
	task.Name = name
	task.Type = "task"
	return task
}
