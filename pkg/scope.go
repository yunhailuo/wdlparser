package wdlparser

import (
	"fmt"
	"path"
	"strings"
)

type scoper interface {
	GetParent() scoper
	SetParent(scoper)
	GetChildren() []scoper
	GetSymbolMap() map[string]symboler
	Define(symboler) error
	Resolve(string) (symboler, error)
}

type scope struct {
	parent    scoper
	children  []scoper
	symbolMap map[string]symboler
}

func newScope() *scope {
	Scope := new(scope)
	Scope.symbolMap = make(map[string]symboler)
	return Scope
}

func (s *scope) GetParent() scoper {
	return s.parent
}

func (s *scope) SetParent(parent scoper) {
	s.parent = parent
}

func (s *scope) GetChildren() []scoper {
	return s.children
}

func (s *scope) GetSymbolMap() map[string]symboler {
	return s.symbolMap
}

func (s *scope) Define(sym symboler) error {
	if sym.GetName() == "" {
		return fmt.Errorf("symbol %v doesn't have a valid name", sym)
	}
	if s.symbolMap == nil {
		return fmt.Errorf("symbolMap of scope %v not initialized", s)
	}
	s.symbolMap[sym.GetName()] = sym
	return nil
}
func (s *scope) Resolve(name string) (symboler, error) {
	if sym, ok := s.symbolMap[name]; ok {
		return sym, nil
	}
	if s.GetParent() != nil {
		return s.GetParent().Resolve(name)
	}
	return nil, fmt.Errorf("%v not defined", name)
}

// WDL represnets a parsed WDL document.
// It is also the global scope of a parsing
type WDL struct {
	scopedSymbol
	Path    string
	Version string
}

func NewWDL(wdlPath string) *WDL {
	wdl := new(WDL)
	wdl.Path = wdlPath
	wdl.SetName(strings.TrimSuffix(path.Base(wdlPath), ".wdl"))
	wdl.SetType("document")
	wdl.symbolMap = make(map[string]symboler)
	return wdl
}

func (wdl WDL) GetImports() map[string]*WDL {
	ret := map[string]*WDL{}
	for k, sym := range wdl.symbolMap {
		if w, ok := sym.(*WDL); ok {
			ret[k] = w
		}
	}
	return ret
}

func (wdl WDL) GetWorkflow() map[string]*Workflow {
	ret := map[string]*Workflow{}
	for k, sym := range wdl.symbolMap {
		if w, ok := sym.(*Workflow); ok {
			ret[k] = w
		}
	}
	return ret
}

func (wdl WDL) GetTask() map[string]*Task {
	ret := map[string]*Task{}
	for k, sym := range wdl.symbolMap {
		if w, ok := sym.(*Task); ok {
			ret[k] = w
		}
	}
	return ret
}

// Workflow records one parsed workflow
type Workflow struct {
	scopedSymbol
	Elements                             []string
	Inputs, Outputs, Meta, ParameterMeta map[string]symboler
}

func NewWorkflow(name string) *Workflow {
	workflow := new(Workflow)
	workflow.SetName(name)
	workflow.SetType("workflow")
	workflow.symbolMap = make(map[string]symboler)
	return workflow
}

// Workflow records one parsed workflow
type Task struct {
	scopedSymbol
	Elements, Command                             []string
	Inputs, Outputs, Runtime, Meta, ParameterMeta map[string]symboler
}

func NewTask(name string) *Task {
	task := new(Task)
	task.SetName(name)
	task.SetType("task")
	task.symbolMap = map[string]symboler{}
	return task
}
