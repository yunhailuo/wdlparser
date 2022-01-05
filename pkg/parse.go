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

// wdlSection describes what WDL entity a node represents.
type wdlSection int

const (
	_   wdlSection = iota // leave 0 as nodeKind zero value; start from 1
	doc                   // WDL document
	imp                   // import
	wfl                   // workflow
	cal                   // call
	tsk                   // task
	ipt                   // input
	opt                   // output
	mtd                   // metadata
	pmt                   // parameter metadata
)

type sectionStack []wdlSection

func (nks *sectionStack) push(nk wdlSection) {
	*nks = append(*nks, nk)
}

func (nks *sectionStack) pop() {
	stackDepth := len(*nks)
	if stackDepth > 0 {
		// Won't zero the popped element since nodeKind is limited and small
		*nks = (*nks)[:stackDepth-1]
		return
	}
	log.Fatalf("pop error: node kind stack %v is empty", *nks)
}

func (nks *sectionStack) contains(nk wdlSection) bool {
	for _, e := range *nks {
		if e == nk {
			return true
		}
	}
	return false
}

type wdlv1_1Listener struct {
	*parser.BaseWdlV1_1ParserListener
	wdl        *WDL
	astContext struct {
		sectionStack sectionStack
		importNode   *importSpec
		workflowNode *Workflow
		callNode     *Call
		taskNode     *Task
		exprRPNStack *exprRPN
	}
}

func newWdlv1_1Listener(wdl *WDL) *wdlv1_1Listener {
	return &wdlv1_1Listener{wdl: wdl}
}

// Manage AST context with listener
func (l *wdlv1_1Listener) EnterEveryRule(ctx antlr.ParserRuleContext) {
	switch c := ctx.(type) {
	// Set up new AST node
	// and point current listener AST context to the new node
	case *parser.DocumentContext:
		// This root context is special
		// as the root AST node is already set up as `l.wdl`
		l.astContext.sectionStack.push(doc)
	case *parser.Import_docContext:
		l.astContext.sectionStack.push(imp)
		n := newImportSpec(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			strings.Trim(c.R_string().GetText(), `"`),
		)
		n.setParent(l.wdl)
		l.wdl.Imports = append(l.wdl.Imports, n)
		l.astContext.importNode = n
	case *parser.WorkflowContext:
		l.astContext.sectionStack.push(wfl)
		l.wdl.Workflow = NewWorkflow(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			c.Identifier().GetText(),
		)
		l.wdl.Workflow.setParent(l.wdl)
		l.astContext.workflowNode = l.wdl.Workflow
	case *parser.CallContext:
		l.astContext.sectionStack.push(cal)
		n := NewCall(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			"",
		)
		n.setParent(l.astContext.workflowNode)
		l.astContext.workflowNode.Calls = append(
			l.astContext.workflowNode.Calls, n,
		)
		l.astContext.callNode = n
	case *parser.TaskContext:
		l.astContext.sectionStack.push(tsk)
		n := NewTask(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			c.Identifier().GetText(),
		)
		n.setParent(l.wdl)
		l.wdl.Tasks = append(l.wdl.Tasks, n)
		l.astContext.taskNode = n

	// No new node is needed
	// but current listener AST context needs to be updated
	case *parser.Workflow_inputContext:
		l.astContext.sectionStack.push(ipt)
	case *parser.Workflow_outputContext:
		l.astContext.sectionStack.push(opt)
	case *parser.Task_inputContext:
		l.astContext.sectionStack.push(ipt)
	case *parser.Task_outputContext:
		l.astContext.sectionStack.push(opt)
	case *parser.Task_runtime_kvContext:
		v := newValueSpec(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			c.Identifier().GetText(),
			"",
		)
		l.astContext.taskNode.Runtime = append(l.astContext.taskNode.Runtime, v)
		l.astContext.exprRPNStack = v.value
	case *parser.MetaContext:
		l.astContext.sectionStack.push(mtd)
	case *parser.Parameter_metaContext:
		l.astContext.sectionStack.push(pmt)
	case *parser.Call_inputContext:
		v := newValueSpec(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			c.Identifier().GetText(),
			"",
		)
		v.name.isReference = true
		l.astContext.callNode.Inputs = append(l.astContext.callNode.Inputs, v)
		l.astContext.exprRPNStack = v.value
	}
}

func (l *wdlv1_1Listener) ExitEveryRule(ctx antlr.ParserRuleContext) {
	switch ctx.(type) {
	case *parser.DocumentContext:
		l.astContext.sectionStack.pop()
	case *parser.Import_docContext:
		l.astContext.sectionStack.pop()
		l.astContext.importNode = nil
	case *parser.WorkflowContext:
		l.astContext.sectionStack.pop()
		l.astContext.workflowNode = nil
	case *parser.CallContext:
		l.astContext.sectionStack.pop()
		l.astContext.callNode = nil
	case *parser.TaskContext:
		l.astContext.sectionStack.pop()
		l.astContext.taskNode = nil

	case *parser.Workflow_inputContext:
		l.astContext.sectionStack.pop()
	case *parser.Workflow_outputContext:
		l.astContext.sectionStack.pop()
	case *parser.Task_inputContext:
		l.astContext.sectionStack.pop()
	case *parser.Task_outputContext:
		l.astContext.sectionStack.pop()
	case *parser.Task_runtime_kvContext:
		l.astContext.exprRPNStack = nil
	case *parser.MetaContext:
		l.astContext.sectionStack.pop()
	case *parser.Parameter_metaContext:
		l.astContext.sectionStack.pop()
	case *parser.Call_inputContext:
		l.astContext.exprRPNStack = nil
	}
}

// Parse WDL version
func (l *wdlv1_1Listener) ExitVersion(ctx *parser.VersionContext) {
	l.wdl.Version = ctx.ReleaseVersion().GetText()
}

// Parse import
func (l *wdlv1_1Listener) ExitImport_as(ctx *parser.Import_asContext) {
	l.astContext.importNode.alias = ctx.Identifier().GetText()
}

func (l *wdlv1_1Listener) ExitImport_alias(ctx *parser.Import_aliasContext) {
	k, v := ctx.Identifier(0).GetText(), ctx.Identifier(1).GetText()
	l.astContext.importNode.importAliases[k] = v
}

// Parse workflow elements
func (l *wdlv1_1Listener) ExitCall_name(ctx *parser.Call_nameContext) {
	l.astContext.callNode.name.initialName = ctx.GetText()
}

func (l *wdlv1_1Listener) ExitCall_alias(ctx *parser.Call_aliasContext) {
	l.astContext.callNode.alias = ctx.Identifier().GetText()
}

func (l *wdlv1_1Listener) ExitCall_after(ctx *parser.Call_afterContext) {
	l.astContext.callNode.After = ctx.Identifier().GetText()
}

// Parse a task
// TODO: wrong parsing to be fixed
func (l *wdlv1_1Listener) EnterTask_command(ctx *parser.Task_commandContext) {
	l.astContext.taskNode.Command = append(
		l.astContext.taskNode.Command,
		ctx.Task_command_string_part().GetText(),
	)
}

// Parse any declaration
func (l *wdlv1_1Listener) EnterUnbound_decls(ctx *parser.Unbound_declsContext) {
	n := newValueSpec(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.Identifier().GetText(),
		ctx.Wdl_type().GetText(),
	)
	// Try to figure out which section this valueSpec belongs to
	switch {
	case l.astContext.sectionStack.contains(wfl):
		l.wdl.Workflow.Inputs = append(l.wdl.Workflow.Inputs, n)
	case l.astContext.sectionStack.contains(tsk):
		taskNode := l.wdl.Tasks[len(l.wdl.Tasks)-1]
		taskNode.Inputs = append(taskNode.Inputs, n)
	default:
		l.wdl.Structs = append(l.wdl.Structs, n)
	}
}

func (l *wdlv1_1Listener) EnterBound_decls(ctx *parser.Bound_declsContext) {
	n := newValueSpec(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.Identifier().GetText(),
		ctx.Wdl_type().GetText(),
	)
	// Try to figure out which section this valueSpec belongs to
	switch {
	case l.astContext.sectionStack.contains(wfl):
		switch {
		case l.astContext.sectionStack.contains(ipt):
			l.wdl.Workflow.Inputs = append(l.wdl.Workflow.Inputs, n)
		case l.astContext.sectionStack.contains(opt):
			l.wdl.Workflow.Outputs = append(l.wdl.Workflow.Outputs, n)
		default:
			l.wdl.Workflow.PrvtDecls = append(l.wdl.Workflow.PrvtDecls, n)
		}
	case l.astContext.sectionStack.contains(tsk):
		taskNode := l.wdl.Tasks[len(l.wdl.Tasks)-1]
		switch {
		case l.astContext.sectionStack.contains(ipt):
			taskNode.Inputs = append(taskNode.Inputs, n)
		case l.astContext.sectionStack.contains(opt):
			taskNode.Outputs = append(taskNode.Outputs, n)
		default:
			taskNode.PrvtDecls = append(taskNode.PrvtDecls, n)
		}
	default:
		l.wdl.Structs = append(l.wdl.Structs, n)
	}
	l.astContext.exprRPNStack = n.value
}

func (l *wdlv1_1Listener) ExitBound_decls(ctx *parser.Bound_declsContext) {
	l.astContext.exprRPNStack = nil
}

// Parse metadata
func (l *wdlv1_1Listener) ExitMeta_kv(ctx *parser.Meta_kvContext) {
	v := newValueSpec(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.MetaIdentifier().GetText(),
		"",
	)
	v.value.append(ctx.Meta_value().GetText())
	switch {
	case l.astContext.sectionStack.contains(wfl):
		switch {
		case l.astContext.sectionStack.contains(mtd):
			l.wdl.Workflow.Meta = append(l.wdl.Workflow.Meta, v)
		case l.astContext.sectionStack.contains(pmt):
			l.wdl.Workflow.ParameterMeta = append(
				l.wdl.Workflow.ParameterMeta, v,
			)
		}
	case l.astContext.sectionStack.contains(tsk):
		taskNode := l.wdl.Tasks[len(l.wdl.Tasks)-1]
		switch {
		case l.astContext.sectionStack.contains(mtd):
			taskNode.Meta = append(taskNode.Meta, v)
		case l.astContext.sectionStack.contains(pmt):
			taskNode.ParameterMeta = append(taskNode.ParameterMeta, v)
		}
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
