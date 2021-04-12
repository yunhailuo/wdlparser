/*
Package wdlparser implements a parser for Workflow Description Language (WDL)
source files. Language specifications can be found at https://github.com/openwdl/wdl
*/
package wdlparser

import (
	"log"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	parser "github.com/yunhailuo/wdlparser/pkg/wdlv1_1"
)

type wdlv1_1Listener struct {
	*parser.BaseWdlV1_1ParserListener
	wdl               *WDL
	currentWDLScope   Scope
	currentANTLRScope Scope
}

func newWdlv1_1Listener(wdlPath string) *wdlv1_1Listener {
	wdl := NewWDL(wdlPath)
	return &wdlv1_1Listener{wdl: wdl}
}

func (l *wdlv1_1Listener) EnterVersion(ctx *parser.VersionContext) {
	l.wdl.Version = ctx.ReleaseVersion().GetText()
}

func (l *wdlv1_1Listener) EnterImport_doc(ctx *parser.Import_docContext) {
	ruleScp := newRuleScope()
	ruleScp.Parent = l.currentANTLRScope
	l.currentANTLRScope = ruleScp
}

func (l *wdlv1_1Listener) EnterImport_as(ctx *parser.Import_asContext) {
	aliasSymbol := Symbol{Name: "alias", Value: ctx.Identifier().GetText()}
	err := l.currentANTLRScope.Define(aliasSymbol)
	if err != nil {
		log.Fatal(err)
	}
}

func (l *wdlv1_1Listener) ExitImport_doc(ctx *parser.Import_docContext) {
	importPath := strings.Trim(ctx.R_string().GetText(), `"`)
	importedWdl := NewImport(importPath)
	aliasSymbol, err := l.currentANTLRScope.Resolve("alias")
	if err != nil {
		l.wdl.Imports[importedWdl.Name] = importedWdl
	} else {
		l.wdl.Imports[aliasSymbol.Value] = importedWdl
	}
	l.currentANTLRScope = l.currentANTLRScope.GetParent()
}

func (l *wdlv1_1Listener) EnterWorkflow(ctx *parser.WorkflowContext) {
	workflow := NewWorkflow(ctx.Identifier().GetText())
	workflow.Parent = l.currentWDLScope
	l.currentWDLScope = workflow
	for _, e := range ctx.AllWorkflow_element() {
		workflow.Elements = append(workflow.Elements, e.GetText())
	}
	l.wdl.Workflow = workflow
}

func (l *wdlv1_1Listener) ExitWorkflow(ctx *parser.WorkflowContext) {
	l.currentWDLScope = l.currentWDLScope.GetParent()
}

func (l *wdlv1_1Listener) EnterTask(ctx *parser.TaskContext) {
	task := NewTask(ctx.Identifier().GetText())
	task.Parent = l.currentWDLScope
	l.currentWDLScope = task
	for _, e := range ctx.AllTask_element() {
		task.Elements = append(task.Elements, e.GetText())
	}
	l.wdl.Tasks[task.Name] = task
}

func (l *wdlv1_1Listener) ExitTask(ctx *parser.TaskContext) {
	l.currentWDLScope = l.currentWDLScope.GetParent()
}

// Antlr4Parse parse a WDL document into WDL
func Antlr4Parse(path string) (*WDL, []WDLSyntaxError) {
	input, err := antlr.NewFileStream(path)
	if err != nil {
		log.Fatal(err)
	}

	lexer := parser.NewWdlV1_1Lexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewWdlV1_1Parser(stream)
	errorListener := newWdlErrorListener(true)
	p.AddErrorListener(errorListener)
	p.BuildParseTrees = true
	listener := newWdlv1_1Listener(path)
	listener.currentWDLScope = listener.wdl
	listener.currentANTLRScope = newRuleScope()
	antlr.ParseTreeWalkerDefault.Walk(listener, p.Document())

	return listener.wdl, errorListener.syntaxErrors
}
