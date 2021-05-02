/*
Package wdlparser implements a parser for Workflow Description Language (WDL)
source files. Language specifications can be found at https://github.com/openwdl/wdl
*/
package wdlparser

import (
	"log"
	"os"
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
	var kind nodeKind = dcl
	switch ctx.GetParent().(type) {
	case *parser.Any_declsContext:
		// any_decls in the current grammar can only be workflow or task inputs
		kind = ipt
	}
	obj := newObject(
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
		if kind == dcl {
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
		if kind == dcl {
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
	var kind nodeKind = dcl
	switch ctx.GetParent().(type) {
	case *parser.Any_declsContext:
		// any_decls in the current grammar can only be workflow or task inputs
		kind = ipt
	case *parser.Workflow_outputContext, *parser.Task_outputContext:
		kind = opt
	}
	obj := newObject(
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
		if kind == dcl {
			n.PrvtDecls = append(n.PrvtDecls, obj)
		} else if kind == ipt {
			n.Inputs = append(n.Inputs, obj)
		} else {
			n.Outputs = append(n.Outputs, obj)
		}
	case *Task:
		if kind == dcl {
			n.PrvtDecls = append(n.PrvtDecls, obj)
		} else if kind == ipt {
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
		kind = mtd
	case *parser.Parameter_metaContext:
		kind = pmt
	default:
		log.Fatalf(
			"Unexpected metadata declaration at line %d:%d"+
				" while in a %T parser context",
			ctx.GetStart().GetLine(),
			ctx.GetStart().GetColumn(),
			c,
		)
	}
	obj := newObject(
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
		if kind == pmt {
			n.ParameterMeta[obj.getName()] = obj
		} else {
			n.Meta[obj.getName()] = obj
		}
	case *Task:
		if kind == pmt {
			n.ParameterMeta[obj.getName()] = obj
		} else {
			n.Meta[obj.getName()] = obj
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
	workflow, ok := l.currentNode.(*Workflow)
	if !ok {
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
			l.wdl.Workflow.getName(),
			workflow.getName(),
		)
	}
	l.wdl.Workflow = workflow
	l.currentNode = l.wdl
}
func (l *wdlv1_1Listener) EnterCall(ctx *parser.CallContext) {
	call := NewCall(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		"",
	)
	call.setParent(l.currentNode)
	l.currentNode = call
}

func (l *wdlv1_1Listener) ExitCall_name(ctx *parser.Call_nameContext) {
	call, ok := l.currentNode.(*Call)
	if !ok {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"call name",
				"call",
				l.currentNode,
			),
		)
	}
	call.setName(ctx.GetText())
}

func (l *wdlv1_1Listener) ExitCall_alias(ctx *parser.Call_aliasContext) {
	call, ok := l.currentNode.(*Call)
	if !ok {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"call alias",
				"call",
				l.currentNode,
			),
		)
	}
	call.setAlias(ctx.Identifier().GetText())
}

func (l *wdlv1_1Listener) ExitCall_after(ctx *parser.Call_afterContext) {
	call, ok := l.currentNode.(*Call)
	if !ok {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"call after",
				"call",
				l.currentNode,
			),
		)
	}
	call.After = ctx.Identifier().GetText()
}

func (l *wdlv1_1Listener) ExitCall_input(ctx *parser.Call_inputContext) {
	call, ok := l.currentNode.(*Call)
	if !ok {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"call input",
				"call",
				l.currentNode,
			),
		)
	}
	call.Inputs = append(
		call.Inputs,
		newObject(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ipt,
			ctx.Identifier().GetText(),
			"",
			ctx.Expr().GetText(),
		),
	)
}

func (l *wdlv1_1Listener) ExitCall(ctx *parser.CallContext) {
	call, currentOk := l.currentNode.(*Call)
	if !currentOk {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"call",
				"call",
				l.currentNode,
			),
		)
	}
	workflow, parentOK := l.currentNode.getParent().(*Workflow)
	if !parentOK {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"call",
				"workflow parent",
				l.currentNode.getParent(),
			),
		)
	}
	workflow.Calls = append(workflow.Calls, call)
	l.currentNode = workflow
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
		t.Runtime[ctx.Identifier().GetText()] = newObject(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			rnt,
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
	task, ok := l.currentNode.(*Task)
	if !ok {
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
func Antlr4Parse(input string) (*WDL, []wdlSyntaxError) {
	inputInfo, err := os.Stat(input)
	var inputStream antlr.CharStream
	var path string = input
	if os.IsNotExist(err) {
		log.Println(
			"Input is not a file path" +
				" so guessing it's a WDL document in string.",
		)
		path = ""
		inputStream = antlr.NewInputStream(input)
	} else if inputInfo.IsDir() {
		log.Fatalf(
			"%v is a directory; need a file path or WDL document string.",
			path,
		)
	} else {
		inputStream, err = antlr.NewFileStream(path)
		if err != nil {
			log.Fatal(err)
		}
	}

	lexer := parser.NewWdlV1_1Lexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewWdlV1_1Parser(stream)
	p.Interpreter.SetPredictionMode(antlr.PredictionModeSLL)
	errorListener := newWdlErrorListener(true)
	p.AddErrorListener(errorListener)
	p.BuildParseTrees = true
	wdl := NewWDL(path, inputStream.Size())
	antlr.ParseTreeWalkerDefault.Walk(newWdlv1_1Listener(wdl), p.Document())

	return wdl, errorListener.syntaxErrors
}
