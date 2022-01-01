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

// This branching method add a new node as a child of current node
// and set current node to the new node
func (l *wdlv1_1Listener) branching(child node, checkout bool) {
	child.setParent(l.currentNode)
	err := l.currentNode.addChild(child)
	if err != nil {
		log.Fatal(err)
	}
	if checkout {
		l.currentNode = child
	}
}

func (l *wdlv1_1Listener) ExitVersion(ctx *parser.VersionContext) {
	l.wdl.Version = ctx.ReleaseVersion().GetText()
}

func (l *wdlv1_1Listener) EnterImport_doc(ctx *parser.Import_docContext) {
	l.branching(
		newImportSpec(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			strings.Trim(ctx.R_string().GetText(), `"`),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitImport_doc(ctx *parser.Import_docContext) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) ExitImport_as(ctx *parser.Import_asContext) {
	currNode, isImportSpec := l.currentNode.(*importSpec)
	if !isImportSpec {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"import statement",
				"importSpec",
				l.currentNode,
			),
		)
	}
	currNode.alias = ctx.Identifier().GetText()
}

func (l *wdlv1_1Listener) ExitImport_alias(ctx *parser.Import_aliasContext) {
	currNode, isImportSpec := l.currentNode.(*importSpec)
	if !isImportSpec {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"import statement",
				"importSpec",
				l.currentNode,
			),
		)
	}
	currNode.importAliases[ctx.Identifier(0).GetText()] = ctx.Identifier(
		1,
	).GetText()

}

func (l *wdlv1_1Listener) EnterUnbound_decls(ctx *parser.Unbound_declsContext) {
	l.branching(
		newDecl(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.Identifier().GetText(),
			ctx.Wdl_type().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitUnbound_decls(ctx *parser.Unbound_declsContext) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterBound_decls(ctx *parser.Bound_declsContext) {
	l.branching(
		newDecl(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.Identifier().GetText(),
			ctx.Wdl_type().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitBound_decls(ctx *parser.Bound_declsContext) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterWorkflow(ctx *parser.WorkflowContext) {
	l.branching(
		NewWorkflow(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.Identifier().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitWorkflow(ctx *parser.WorkflowContext) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterWorkflow_input(
	ctx *parser.Workflow_inputContext,
) {
	l.branching(
		newInputDecls(ctx.GetStart().GetLine(), ctx.GetStart().GetColumn()),
		true,
	)
}

func (l *wdlv1_1Listener) ExitWorkflow_input(
	ctx *parser.Workflow_inputContext,
) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterCall(ctx *parser.CallContext) {
	l.branching(
		NewCall(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			"",
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitCall(ctx *parser.CallContext) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) ExitCall_name(ctx *parser.Call_nameContext) {
	call, isCall := l.currentNode.(*Call)
	if !isCall {
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
	call.name = ctx.GetText()
}

func (l *wdlv1_1Listener) ExitCall_alias(ctx *parser.Call_aliasContext) {
	call, isCall := l.currentNode.(*Call)
	if !isCall {
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
	call.alias = ctx.Identifier().GetText()
}

func (l *wdlv1_1Listener) ExitCall_after(ctx *parser.Call_afterContext) {
	call, isCall := l.currentNode.(*Call)
	if !isCall {
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
	call, isCall := l.currentNode.(*Call)
	if !isCall {
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
		newKeyValue(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.Identifier().GetText(),
			ctx.Expr().GetText(),
		),
	)
}

func (l *wdlv1_1Listener) EnterWorkflow_output(
	ctx *parser.Workflow_outputContext,
) {
	l.branching(
		newOutputDecls(ctx.GetStart().GetLine(), ctx.GetStart().GetColumn()),
		true,
	)
}

func (l *wdlv1_1Listener) ExitWorkflow_output(
	ctx *parser.Workflow_outputContext,
) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterMeta(ctx *parser.MetaContext) {
	l.branching(
		newMetaSpecs(ctx.GetStart().GetLine(), ctx.GetStart().GetColumn()),
		true,
	)
}

func (l *wdlv1_1Listener) ExitMeta(ctx *parser.MetaContext) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterParameter_meta(
	ctx *parser.Parameter_metaContext,
) {
	l.branching(
		newParameterMetaSpecs(
			ctx.GetStart().GetLine(), ctx.GetStart().GetColumn(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitParameter_meta(
	ctx *parser.Parameter_metaContext,
) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterMeta_kv(ctx *parser.Meta_kvContext) {
	switch n := l.currentNode.(type) {
	case *metaSpec:
		n.keyValues[ctx.MetaIdentifier().GetText()] = ctx.Meta_value().GetText()
	case *parameterMetaSpec:
		n.keyValues[ctx.MetaIdentifier().GetText()] = ctx.Meta_value().GetText()
	default:
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"metadata key/value pairs",
				"metaSpec or parameterMetaSpec",
				l.currentNode.getParent(),
			),
		)
	}
}

func (l *wdlv1_1Listener) EnterTask(ctx *parser.TaskContext) {
	l.branching(
		NewTask(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.Identifier().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitTask(ctx *parser.TaskContext) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterTask_input(
	ctx *parser.Task_inputContext,
) {
	l.branching(
		newInputDecls(ctx.GetStart().GetLine(), ctx.GetStart().GetColumn()),
		true,
	)
}

func (l *wdlv1_1Listener) ExitTask_input(
	ctx *parser.Task_inputContext,
) {
	l.currentNode = l.currentNode.getParent()
}

// TODO: wrong parsing to be fixed
func (l *wdlv1_1Listener) EnterTask_command(ctx *parser.Task_commandContext) {
	if task, isTask := l.currentNode.(*Task); isTask {
		task.Command = append(
			task.Command, ctx.Task_command_string_part().GetText(),
		)
	}
}

// TODO: wrong parsing to be fixed
func (l *wdlv1_1Listener) ExitTask_command_expr_with_string(
	ctx *parser.Task_command_expr_with_stringContext,
) {
	if task, isTask := l.currentNode.(*Task); isTask {
		task.Command = append(
			task.Command,
			ctx.Task_command_expr_part().GetText(),
			ctx.Task_command_string_part().GetText(),
		)
	}
}

func (l *wdlv1_1Listener) EnterTask_output(
	ctx *parser.Task_outputContext,
) {
	l.branching(
		newOutputDecls(ctx.GetStart().GetLine(), ctx.GetStart().GetColumn()),
		true,
	)
}

func (l *wdlv1_1Listener) ExitTask_output(
	ctx *parser.Task_outputContext,
) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterTask_runtime(ctx *parser.Task_runtimeContext) {
	l.branching(
		newRuntimeSpecs(ctx.GetStart().GetLine(), ctx.GetStart().GetColumn()),
		true,
	)
}

func (l *wdlv1_1Listener) ExitTask_runtime(ctx *parser.Task_runtimeContext) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) ExitTask_runtime_kv(
	ctx *parser.Task_runtime_kvContext,
) {
	if r, isRuntime := l.currentNode.(*runtimeSpec); isRuntime {
		r.keyValues[ctx.Identifier().GetText()] = ctx.Expr().GetText()
	} else {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"task runtime key/value pairs",
				"runtimeSpec",
				l.currentNode,
			),
		)
	}
}

// Antlr4Parse parse a WDL document into WDL
func Antlr4Parse(input string) (*WDL, []wdlSyntaxError) {
	inputInfo, err := os.Stat(input)
	var inputStream antlr.CharStream
	var path string = input
	if err != nil {
		log.Println(
			"Input is not a valid file path" +
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
	p.BuildParseTrees = false
	p.Interpreter.SetPredictionMode(antlr.PredictionModeSLL)
	errorListener := newWdlErrorListener(true)
	p.AddErrorListener(errorListener)
	p.BuildParseTrees = true
	wdl := NewWDL(path, inputStream.Size())
	antlr.ParseTreeWalkerDefault.Walk(newWdlv1_1Listener(wdl), p.Document())

	return wdl, errorListener.syntaxErrors
}
