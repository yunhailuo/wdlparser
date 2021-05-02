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
	wdl         *WDL
	currentNode node
}

func newWdlv1_1Listener(wdl *WDL) *wdlv1_1Listener {
	return &wdlv1_1Listener{wdl: wdl, currentNode: wdl}
}

func (l *wdlv1_1Listener) ExitVersion(ctx *parser.VersionContext) {
	l.wdl.Version = ctx.ReleaseVersion().GetText()
}

func (l *wdlv1_1Listener) ExitImport_doc(ctx *parser.Import_docContext) {
	// Build an import node
	importPath := strings.Trim(ctx.R_string().GetText(), `"`)
	importedWdl := NewWDL(importPath, 0)
	importedWdl.setKind(imp)
	for _, child := range ctx.GetChildren() {
		switch childCtx := child.(type) {
		case *parser.Import_asContext:
			importedWdl.setAlias(childCtx.Identifier().GetText())
		}
	}
	// Put the import node into AST
	l.wdl.Imports = append(l.wdl.Imports, importedWdl)
}

func (l *wdlv1_1Listener) ExitUnbound_decls(ctx *parser.Unbound_declsContext) {
	// Build a declaration or input object
	var kind nodeKind = Dcl
	switch ctx.GetParent().(type) {
	case *parser.Any_declsContext:
		// any_decls in the current grammar can only be workflow or task inputs
		kind = Ipt
	}
	obj := NewObject(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		kind,
		ctx.Identifier().GetText(),
		ctx.Wdl_type().GetText(),
		"",
	)
	// Put the object into AST
	switch n := l.currentNode.(type) {
	case *Workflow:
		if kind == Dcl {
			log.Fatalf(
				"Line %d:%d: found a private unbound declaration"+
					" while in a workflow listener node",
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
			)
		} else {
			n.Inputs = append(n.Inputs, obj)
		}
	case *Task:
		if kind == Dcl {
			log.Fatalf(
				"Line %d:%d: found a private unbound declaration"+
					" while in a task listener node",
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
			)
		} else {
			n.Inputs = append(n.Inputs, obj)
		}
	default:
		log.Fatalf(
			"Unexpected unbound declaration at line %d:%d"+
				" while in a %T listener node",
			ctx.GetStart().GetLine(),
			ctx.GetStart().GetColumn(),
			n,
		)
	}
}

func (l *wdlv1_1Listener) ExitBound_decls(ctx *parser.Bound_declsContext) {
	// Build an input, output or declaration object
	var kind nodeKind = Dcl
	switch ctx.GetParent().(type) {
	case *parser.Any_declsContext:
		// any_decls in the current grammar can only be workflow or task inputs
		kind = Ipt
	case *parser.Workflow_outputContext, *parser.Task_outputContext:
		kind = Opt
	}
	obj := NewObject(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		kind,
		ctx.Identifier().GetText(),
		ctx.Wdl_type().GetText(),
		ctx.Expr().GetText(),
	)
	// Put the object into AST
	switch n := l.currentNode.(type) {
	case *Workflow:
		if kind == Dcl {
			n.PrvtDecls = append(n.PrvtDecls, obj)
		} else if kind == Ipt {
			n.Inputs = append(n.Inputs, obj)
		} else {
			n.Outputs = append(n.Outputs, obj)
		}
	case *Task:
		if kind == Dcl {
			n.PrvtDecls = append(n.PrvtDecls, obj)
		} else if kind == Ipt {
			n.Inputs = append(n.Inputs, obj)
		} else {
			n.Outputs = append(n.Outputs, obj)
		}
	default:
		log.Fatalf(
			"Unexpected bound declaration at line %d:%d"+
				" while in a %T listener node",
			ctx.GetStart().GetLine(),
			ctx.GetStart().GetColumn(),
			n,
		)
	}
}

func (l *wdlv1_1Listener) EnterMeta_kv(ctx *parser.Meta_kvContext) {
	// Build a metadata or parameter metadata object
	var kind nodeKind
	switch c := ctx.GetParent().(type) {
	case *parser.MetaContext:
		kind = Mtd
	case *parser.Parameter_metaContext:
		kind = Pmt
	default:
		log.Fatalf(
			"Unexpected metadata declaration at line %d:%d"+
				" while in a %T parser context",
			ctx.GetStart().GetLine(),
			ctx.GetStart().GetColumn(),
			c,
		)
	}
	obj := NewObject(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		kind,
		ctx.MetaIdentifier().GetText(),
		"",
		ctx.Meta_value().GetText(),
	)
	// Put the object into AST
	switch n := l.currentNode.(type) {
	case *Workflow:
		if kind == Pmt {
			n.ParameterMeta[obj.GetName()] = obj
		} else {
			n.Meta[obj.GetName()] = obj
		}
	case *Task:
		if kind == Pmt {
			n.ParameterMeta[obj.GetName()] = obj
		} else {
			n.Meta[obj.GetName()] = obj
		}
	default:
		log.Fatalf(
			"Unexpected metadata declaration at line %d:%d"+
				" while in a %T listener node",
			ctx.GetStart().GetLine(),
			ctx.GetStart().GetColumn(),
			n,
		)
	}
}

func (l *wdlv1_1Listener) EnterWorkflow(ctx *parser.WorkflowContext) {
	workflow := NewWorkflow(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.Identifier().GetText(),
	)
	workflow.setParent(l.currentNode)
	l.currentNode = workflow
	for _, e := range ctx.AllWorkflow_element() {
		workflow.Elements = append(workflow.Elements, e.GetText())
	}
}

func (l *wdlv1_1Listener) ExitWorkflow(ctx *parser.WorkflowContext) {
	workflow, currentOk := l.currentNode.(*Workflow)
	if !currentOk {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"workflow",
				"workflow",
				l.currentNode,
			),
		)
	}
	if l.wdl.Workflow != nil {
		log.Fatalf(
			"Found a \"%v\" workflow while a \"%v\" workflow already exists;"+
				" a maximum of one workflow is allowed by grammar"+
				" so this is likely a parsing error",
			l.wdl.Workflow.GetName(),
			workflow.GetName(),
		)
	}
	l.wdl.Workflow = workflow
	l.currentNode = l.wdl
}

func (l *wdlv1_1Listener) EnterTask(ctx *parser.TaskContext) {
	task := NewTask(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.Identifier().GetText(),
	)
	task.setParent(l.currentNode)
	l.currentNode = task
}

func (l *wdlv1_1Listener) EnterTask_command(ctx *parser.Task_commandContext) {
	if task, ok := l.currentNode.(*Task); ok {
		task.Command = append(
			task.Command, ctx.Task_command_string_part().GetText(),
		)
	}
}

func (l *wdlv1_1Listener) ExitTask_command_expr_with_string(
	ctx *parser.Task_command_expr_with_stringContext,
) {
	if task, ok := l.currentNode.(*Task); ok {
		task.Command = append(
			task.Command,
			ctx.Task_command_expr_part().GetText(),
			ctx.Task_command_string_part().GetText(),
		)
	}
}

func (l *wdlv1_1Listener) ExitTask_runtime_kv(
	ctx *parser.Task_runtime_kvContext,
) {
	if t, ok := l.currentNode.(*Task); ok {
		t.Runtime[ctx.Identifier().GetText()] = NewObject(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			Rnt,
			ctx.Identifier().GetText(),
			"",
			ctx.Expr().GetText(),
		)
	} else {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"task runtime",
				"task",
				l.currentNode,
			),
		)
	}
}

func (l *wdlv1_1Listener) ExitTask(ctx *parser.TaskContext) {
	task, currentOk := l.currentNode.(*Task)
	if !currentOk {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"task",
				"task",
				l.currentNode,
			),
		)
	}
	// TODO: check task name is unique in a WDL document
	l.wdl.Tasks = append(l.wdl.Tasks, task)
	l.currentNode = l.wdl
}

// Antlr4Parse parse a WDL document into WDL
func Antlr4Parse(path string) (*WDL, []wdlSyntaxError) {
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
	wdl := NewWDL(path, input.Size())
	antlr.ParseTreeWalkerDefault.Walk(newWdlv1_1Listener(wdl), p.Document())

	return wdl, errorListener.syntaxErrors
}
