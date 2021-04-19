/*
Package wdlparser implements a parser for Workflow Description Language (WDL)
source files. Language specifications can be found at https://github.com/openwdl/wdl
*/
package wdlparser

import (
	"fmt"
	"log"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	parser "github.com/yunhailuo/wdlparser/pkg/wdlv1_1"
)

type wdlv1_1Listener struct {
	*parser.BaseWdlV1_1ParserListener
	wdl          *WDL
	currentScope Scoper
}

func newWdlv1_1Listener(wdlPath string) *wdlv1_1Listener {
	wdl := NewWDL(wdlPath)
	return &wdlv1_1Listener{wdl: wdl}
}

func (l *wdlv1_1Listener) EnterVersion(ctx *parser.VersionContext) {
	l.wdl.Version = ctx.ReleaseVersion().GetText()
}

func (l *wdlv1_1Listener) EnterImport_doc(ctx *parser.Import_docContext) {
	importPath := strings.Trim(ctx.R_string().GetText(), `"`)
	importedWdl := NewImport(importPath)
	importedWdl.SetParent(l.currentScope)
	l.currentScope = importedWdl
}

func (l *wdlv1_1Listener) ExitImport_as(ctx *parser.Import_asContext) {
	if importScope, ok := l.currentScope.(*Import); ok {
		importScope.Alias = ctx.Identifier().GetText()
	} else {
		ctx.GetParser().NotifyErrorListeners(
			`extraneous "import as" outside import statements`,
			ctx.GetStart(),
			nil,
		)
	}
}

func (l *wdlv1_1Listener) ExitImport_doc(ctx *parser.Import_docContext) {
	if importScope, ok := l.currentScope.(*Import); ok {
		if importScope.Alias != "" {
			l.wdl.Imports[importScope.Alias] = importScope
		} else {
			l.wdl.Imports[importScope.Name] = importScope
		}
	} else {
		log.Fatal(
			fmt.Sprintf(
				"Wrong scope at line %d:%d: expecting an import scope",
				ctx.GetStart().GetLine(), ctx.GetStart().GetColumn(),
			),
		)
	}
	l.currentScope = l.currentScope.GetParent()
}

func (l *wdlv1_1Listener) EnterWorkflow(ctx *parser.WorkflowContext) {
	workflow := NewWorkflow(ctx.Identifier().GetText())
	workflow.SetParent(l.currentScope)
	l.currentScope = workflow
	for _, e := range ctx.AllWorkflow_element() {
		workflow.Elements = append(workflow.Elements, e.GetText())
	}
}

func (l *wdlv1_1Listener) ExitWorkflow(ctx *parser.WorkflowContext) {
	if workflowScope, ok := l.currentScope.(*Workflow); ok {
		l.wdl.Workflow = workflowScope
	}
	l.currentScope = l.currentScope.GetParent()
}

func (l *wdlv1_1Listener) EnterTask(ctx *parser.TaskContext) {
	task := NewTask(ctx.Identifier().GetText())
	task.SetParent(l.currentScope)
	l.currentScope = task
	for _, e := range ctx.AllTask_element() {
		task.Elements = append(task.Elements, e.GetText())
	}
}

func (l *wdlv1_1Listener) ExitTask(ctx *parser.TaskContext) {
	if taskScope, ok := l.currentScope.(*Task); ok {
		l.wdl.Tasks[taskScope.Name] = taskScope
	}
	l.currentScope = l.currentScope.GetParent()
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
	listener.currentScope = listener.wdl
	antlr.ParseTreeWalkerDefault.Walk(listener, p.Document())

	return listener.wdl, errorListener.syntaxErrors
}
