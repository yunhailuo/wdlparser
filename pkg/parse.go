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
	currentScope namespace
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
	importedWdl := NewWDL(importPath)
	importedWdl.setKind(imp)
	importedWdl.setParent(l.currentScope)
	l.currentScope = importedWdl
}

func (l *wdlv1_1Listener) ExitImport_as(ctx *parser.Import_asContext) {
	if importScope, ok := l.currentScope.(*WDL); ok {
		importScope.setAlias(ctx.Identifier().GetText())
	} else {
		ctx.GetParser().NotifyErrorListeners(
			`extraneous "import as" outside WDL import statements`,
			ctx.GetStart(),
			nil,
		)
	}
}

func (l *wdlv1_1Listener) ExitImport_doc(ctx *parser.Import_docContext) {
	parentScope := l.currentScope.getParent()
	importedWdl, ok := l.currentScope.(*WDL)
	if (parentScope == nil) || !ok {
		log.Fatal(
			fmt.Sprintf(
				"Wrong scope at line %d:%d: expecting a nested import scope",
				ctx.GetStart().GetLine(), ctx.GetStart().GetColumn(),
			),
		)
	} else {
		parentScope.addDeclaration(importedWdl)
	}
	l.currentScope = l.currentScope.getParent()
}

func (l *wdlv1_1Listener) ExitUnbound_decls(ctx *parser.Unbound_declsContext) {
	l.currentScope.addDeclaration(
		NewObject(
			Dcl, ctx.Identifier().GetText(), ctx.Wdl_type().GetText(), "",
		),
	)
}

func (l *wdlv1_1Listener) ExitBound_decls(ctx *parser.Bound_declsContext) {
	l.currentScope.addDeclaration(
		NewObject(
			Dcl,
			ctx.Identifier().GetText(),
			ctx.Wdl_type().GetText(),
			ctx.Expr().GetText(),
		),
	)
}

func (l *wdlv1_1Listener) EnterMeta_kv(ctx *parser.Meta_kvContext) {
	l.currentScope.addDeclaration(
		NewObject(
			Mtd,
			ctx.MetaIdentifier().GetText(),
			"",
			ctx.Meta_value().GetText(),
		),
	)
}

func (l *wdlv1_1Listener) EnterMeta(ctx *parser.MetaContext) {
	scp := newScope()
	scp.setParent(l.currentScope)
	l.currentScope = scp
}

func (l *wdlv1_1Listener) ExitMeta(ctx *parser.MetaContext) {
	for _, d := range l.currentScope.getDeclarations() {
		d.setKind(Mtd)
		l.currentScope.getParent().addDeclaration(d)
	}
	l.currentScope = l.currentScope.getParent()
}

func (l *wdlv1_1Listener) EnterParameter_meta(ctx *parser.Parameter_metaContext) {
	scp := newScope()
	scp.setParent(l.currentScope)
	l.currentScope = scp
}

func (l *wdlv1_1Listener) ExitParameter_meta(ctx *parser.Parameter_metaContext) {
	for _, d := range l.currentScope.getDeclarations() {
		d.setKind(Pmt)
		l.currentScope.getParent().addDeclaration(d)
	}
	l.currentScope = l.currentScope.getParent()
}

func (l *wdlv1_1Listener) EnterWorkflow(ctx *parser.WorkflowContext) {
	workflow := NewWorkflow(ctx.Identifier().GetText())
	workflow.setParent(l.currentScope)
	l.currentScope = workflow
	for _, e := range ctx.AllWorkflow_element() {
		workflow.Elements = append(workflow.Elements, e.GetText())
	}
}

func (l *wdlv1_1Listener) EnterWorkflow_input(ctx *parser.Workflow_inputContext) {
	scp := newScope()
	scp.setParent(l.currentScope)
	l.currentScope = scp
}

func (l *wdlv1_1Listener) ExitWorkflow_input(ctx *parser.Workflow_inputContext) {
	for _, d := range l.currentScope.getDeclarations() {
		d.setKind(Ipt)
		l.currentScope.getParent().addDeclaration(d)
	}
	l.currentScope = l.currentScope.getParent()
}

func (l *wdlv1_1Listener) EnterWorkflow_output(ctx *parser.Workflow_outputContext) {
	scp := newScope()
	scp.setParent(l.currentScope)
	l.currentScope = scp
}

func (l *wdlv1_1Listener) ExitWorkflow_output(ctx *parser.Workflow_outputContext) {
	for _, d := range l.currentScope.getDeclarations() {
		d.setKind(Opt)
		l.currentScope.getParent().addDeclaration(d)
	}
	l.currentScope = l.currentScope.getParent()
}

func (l *wdlv1_1Listener) ExitWorkflow(ctx *parser.WorkflowContext) {
	parentScope := l.currentScope.getParent()
	workflow, ok := l.currentScope.(*Workflow)
	if (parentScope == nil) || !ok {
		log.Fatal(
			fmt.Sprintf(
				"Wrong scope at line %d:%d: expecting a nested workflow scope",
				ctx.GetStart().GetLine(), ctx.GetStart().GetColumn(),
			),
		)
	} else {
		parentScope.addDeclaration(workflow)
	}
	l.currentScope = parentScope
}

func (l *wdlv1_1Listener) EnterTask(ctx *parser.TaskContext) {
	task := NewTask(ctx.Identifier().GetText())
	task.setParent(l.currentScope)
	l.currentScope = task
	for _, e := range ctx.AllTask_element() {
		task.Elements = append(task.Elements, e.GetText())
	}
}

func (l *wdlv1_1Listener) EnterTask_input(ctx *parser.Task_inputContext) {
	scp := newScope()
	scp.setParent(l.currentScope)
	l.currentScope = scp
}

func (l *wdlv1_1Listener) ExitTask_input(ctx *parser.Task_inputContext) {
	for _, d := range l.currentScope.getDeclarations() {
		d.setKind(Ipt)
		l.currentScope.getParent().addDeclaration(d)
	}
	l.currentScope = l.currentScope.getParent()
}

func (l *wdlv1_1Listener) EnterTask_command(ctx *parser.Task_commandContext) {
	if task, ok := l.currentScope.(*Task); ok {
		task.Command = append(
			task.Command, ctx.Task_command_string_part().GetText(),
		)
	}
}

func (l *wdlv1_1Listener) ExitTask_command_expr_with_string(
	ctx *parser.Task_command_expr_with_stringContext,
) {
	if task, ok := l.currentScope.(*Task); ok {
		task.Command = append(
			task.Command,
			ctx.Task_command_expr_part().GetText(),
			ctx.Task_command_string_part().GetText(),
		)
	}
}

func (l *wdlv1_1Listener) EnterTask_output(ctx *parser.Task_outputContext) {
	scp := newScope()
	scp.setParent(l.currentScope)
	l.currentScope = scp
}

func (l *wdlv1_1Listener) ExitTask_output(ctx *parser.Task_outputContext) {
	for _, d := range l.currentScope.getDeclarations() {
		d.setKind(Opt)
		l.currentScope.getParent().addDeclaration(d)
	}
	l.currentScope = l.currentScope.getParent()
}

func (l *wdlv1_1Listener) ExitTask_runtime_kv(
	ctx *parser.Task_runtime_kvContext,
) {
	l.currentScope.addDeclaration(
		NewObject(
			Rnt,
			ctx.Identifier().GetText(),
			"",
			ctx.Expr().GetText(),
		),
	)
}

func (l *wdlv1_1Listener) ExitTask(ctx *parser.TaskContext) {
	parentScope := l.currentScope.getParent()
	task, ok := l.currentScope.(*Task)
	if (parentScope == nil) || !ok {
		log.Fatal(
			fmt.Sprintf(
				"Wrong scope at line %d:%d: expecting a nested task scope",
				ctx.GetStart().GetLine(), ctx.GetStart().GetColumn(),
			),
		)
	} else {
		parentScope.addDeclaration(task)
	}
	l.currentScope = parentScope
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
	p.Interpreter.SetPredictionMode(antlr.PredictionModeSLL)
	errorListener := newWdlErrorListener(true)
	p.AddErrorListener(errorListener)
	p.BuildParseTrees = true
	listener := newWdlv1_1Listener(path)
	listener.currentScope = listener.wdl
	antlr.ParseTreeWalkerDefault.Walk(listener, p.Document())

	return listener.wdl, errorListener.syntaxErrors
}
