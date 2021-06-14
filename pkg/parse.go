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
	l.currentNode.addChild(child)
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
	l.wdl.Imports = append(l.wdl.Imports, currNode)
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
			"",
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
			ctx.Expr().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitBound_decls(ctx *parser.Bound_declsContext) {
	obj, isDecl := l.currentNode.(*decl)
	if !isDecl {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"bound declaration",
				"declaration",
				l.currentNode,
			),
		)
	}
	var evals []evaluator
	for _, child := range obj.getChildren() {
		if e, isExpr := child.(evaluator); isExpr {
			evals = append(evals, e)
		}
	}
	evalCount := len(evals)
	if evalCount > 1 {
		log.Fatalf(
			"Parser error: bound declaration can have only one child"+
				" expressions, found %v",
			evalCount,
		)
	}
	if evalCount == 1 {
		obj.evaluator = evals[0]
	}
	l.currentNode = l.currentNode.getParent()
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
	obj := newKeyValue(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.MetaIdentifier().GetText(),
		ctx.Meta_value().GetText(),
	)
	// Put the object into AST
	switch n := l.currentNode.(type) {
	case *Workflow:
		if kind == pmt {
			n.ParameterMeta[obj.key] = obj
		} else {
			n.Meta[obj.key] = obj
		}
	case *Task:
		if kind == pmt {
			n.ParameterMeta[obj.key] = obj
		} else {
			n.Meta[obj.key] = obj
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
	l.branching(workflow, true)
	for _, e := range ctx.AllWorkflow_element() {
		workflow.Elements = append(workflow.Elements, e.GetText())
	}
}

func (l *wdlv1_1Listener) ExitWorkflow(ctx *parser.WorkflowContext) {
	workflow, isWorkflow := l.currentNode.(*Workflow)
	if !isWorkflow {
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
	// Collect and organize private declarations
	for _, child := range workflow.getChildren() {
		if d, isDecl := child.(*decl); isDecl {
			workflow.PrvtDecls = append(workflow.PrvtDecls, d)
		}
	}
	if l.wdl.Workflow != nil {
		log.Fatalf(
			"Found a \"%v\" workflow while a \"%v\" workflow already exists;"+
				" a maximum of one workflow is allowed by grammar"+
				" so this is likely a parsing error",
			l.wdl.Workflow.name, workflow.name,
		)
	}
	l.wdl.Workflow = workflow
	l.currentNode = l.wdl
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
	inputs, currentIsInputDecls := l.currentNode.(*inputDecls)
	if !currentIsInputDecls {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"workflow input",
				"inputDecls",
				l.currentNode,
			),
		)
	}
	workflow, parentIsWorkflow := l.currentNode.getParent().(*Workflow)
	if !parentIsWorkflow {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"workflow input",
				"workflow parent",
				l.currentNode.getParent(),
			),
		)
	}
	workflow.Inputs = inputs
	l.currentNode = workflow
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
	call, currentIsCall := l.currentNode.(*Call)
	if !currentIsCall {
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
	workflow, parentIsWorkflow := l.currentNode.getParent().(*Workflow)
	if !parentIsWorkflow {
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
	outputs, currentIsOutputDecls := l.currentNode.(*outputDecls)
	if !currentIsOutputDecls {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"workflow output",
				"outputDecls",
				l.currentNode,
			),
		)
	}
	workflow, parentIsWorkflow := l.currentNode.getParent().(*Workflow)
	if !parentIsWorkflow {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"workflow output",
				"workflow parent",
				l.currentNode.getParent(),
			),
		)
	}
	workflow.Outputs = outputs
	l.currentNode = workflow
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
	task, isTask := l.currentNode.(*Task)
	if !isTask {
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
	// Collect and organize private declarations
	for _, child := range task.getChildren() {
		if d, isDecl := child.(*decl); isDecl {
			task.PrvtDecls = append(task.PrvtDecls, d)
		}
	}
	// TODO: check task name is unique in a WDL document
	l.wdl.Tasks = append(l.wdl.Tasks, task)
	l.currentNode = l.wdl
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
	inputs, currentIsInputDecls := l.currentNode.(*inputDecls)
	if !currentIsInputDecls {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"task input",
				"inputDecls",
				l.currentNode,
			),
		)
	}
	task, parentIsTask := l.currentNode.getParent().(*Task)
	if !parentIsTask {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"task input",
				"task parent",
				l.currentNode.getParent(),
			),
		)
	}
	task.Inputs = inputs
	l.currentNode = task
}

func (l *wdlv1_1Listener) EnterTask_command(ctx *parser.Task_commandContext) {
	if task, isTask := l.currentNode.(*Task); isTask {
		task.Command = append(
			task.Command, ctx.Task_command_string_part().GetText(),
		)
	}
}

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
	outputs, currentIsOutputDecls := l.currentNode.(*outputDecls)
	if !currentIsOutputDecls {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"task output",
				"outputDecls",
				l.currentNode,
			),
		)
	}
	task, parentIsTask := l.currentNode.getParent().(*Task)
	if !parentIsTask {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"task output",
				"task parent",
				l.currentNode.getParent(),
			),
		)
	}
	task.Outputs = outputs
	l.currentNode = task
}

func (l *wdlv1_1Listener) ExitTask_runtime_kv(
	ctx *parser.Task_runtime_kvContext,
) {
	if t, isTask := l.currentNode.(*Task); isTask {
		t.Runtime[ctx.Identifier().GetText()] = newKeyValue(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.Identifier().GetText(),
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
	p.Interpreter.SetPredictionMode(antlr.PredictionModeSLL)
	errorListener := newWdlErrorListener(true)
	p.AddErrorListener(errorListener)
	p.BuildParseTrees = true
	wdl := NewWDL(path, inputStream.Size())
	antlr.ParseTreeWalkerDefault.Walk(newWdlv1_1Listener(wdl), p.Document())

	return wdl, errorListener.syntaxErrors
}
