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

type nodeKindStack []nodeKind

func (nks *nodeKindStack) push(nk nodeKind) {
	*nks = append(*nks, nk)
}

func (nks *nodeKindStack) pop() {
	stackDepth := len(*nks)
	if stackDepth > 0 {
		// Won't zero the popped element since nodeKind is limited and small
		*nks = (*nks)[:stackDepth-1]
		return
	}
	log.Fatalf("pop error: node kind stack %v is empty", *nks)
}

func (nks *nodeKindStack) contains(nk nodeKind) bool {
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
		kindStack       nodeKindStack
		importNode      *importSpec
		workflowNode    *Workflow
		callNode        *Call
		taskNode        *Task
		declarationList *[]*decl
		exprNode        *exprRPN
		metadataList    *map[string]string
	}
}

func newWdlv1_1Listener(wdl *WDL) *wdlv1_1Listener {
	return &wdlv1_1Listener{wdl: wdl}
}

// Manage AST context with listener
func (l *wdlv1_1Listener) EnterEveryRule(ctx antlr.ParserRuleContext) {
	switch c := ctx.(type) {
	case *parser.DocumentContext:
		l.astContext.kindStack.push(doc)
	case *parser.Document_elementContext:
		l.astContext.declarationList = &l.wdl.Structs
	case *parser.Import_docContext:
		l.astContext.kindStack.push(imp)
		n := newImportSpec(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			strings.Trim(c.R_string().GetText(), `"`),
		)
		n.setParent(l.wdl)
		l.wdl.Imports = append(l.wdl.Imports, n)
		l.astContext.importNode = n
	case *parser.WorkflowContext:
		l.astContext.kindStack.push(wfl)
		l.wdl.Workflow = NewWorkflow(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			c.Identifier().GetText(),
		)
		l.wdl.Workflow.setParent(l.wdl)
		l.astContext.workflowNode = l.wdl.Workflow
	case *parser.Workflow_inputContext:
		l.astContext.kindStack.push(ipt)
		l.astContext.declarationList = &l.astContext.workflowNode.Inputs
	case *parser.Workflow_outputContext:
		l.astContext.kindStack.push(opt)
		l.astContext.declarationList = &l.astContext.workflowNode.Outputs
	case *parser.Inner_workflow_elementContext:
		l.astContext.declarationList = &l.astContext.workflowNode.PrvtDecls
	case *parser.CallContext:
		l.astContext.kindStack.push(cal)
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
		l.astContext.kindStack.push(tsk)
		n := NewTask(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			c.Identifier().GetText(),
		)
		n.setParent(l.wdl)
		l.wdl.Tasks = append(l.wdl.Tasks, n)
		l.astContext.taskNode = n
		l.astContext.declarationList = &l.astContext.taskNode.PrvtDecls
	case *parser.Task_inputContext:
		l.astContext.kindStack.push(ipt)
		l.astContext.declarationList = &l.astContext.taskNode.Inputs
	case *parser.Task_outputContext:
		l.astContext.kindStack.push(opt)
		l.astContext.declarationList = &l.astContext.taskNode.Outputs
	case *parser.Task_runtime_kvContext:
		k := c.Identifier().GetText()
		v := &exprRPN{}
		l.astContext.taskNode.Runtime[newIdentifier(k, false)] = v
		l.astContext.exprNode = v
	case *parser.MetaContext:
		l.astContext.kindStack.push(mtd)
		if l.astContext.kindStack.contains(wfl) {
			l.astContext.metadataList = &l.astContext.workflowNode.Meta
		} else if l.astContext.kindStack.contains(tsk) {
			l.astContext.metadataList = &l.astContext.taskNode.Meta
		} else {
			log.Fatalf(
				"enter metadata parser context %v"+
					" while AST context is outside workflow or task", c,
			)
		}
	case *parser.Parameter_metaContext:
		l.astContext.kindStack.push(pmt)
		if l.astContext.kindStack.contains(wfl) {
			l.astContext.metadataList = &l.astContext.workflowNode.ParameterMeta
		} else if l.astContext.kindStack.contains(tsk) {
			l.astContext.metadataList = &l.astContext.taskNode.ParameterMeta
		} else {
			log.Fatalf(
				"enter parameter metadata parser context %v"+
					" while AST context is outside workflow or task", c,
			)
		}
	case *parser.Bound_declsContext:
		n := newDecl(
			c.GetStart().GetStart(),
			c.GetStop().GetStop(),
			c.Identifier().GetText(),
			c.Wdl_type().GetText(),
		)
		l.astContext.exprNode = &n.value
		*l.astContext.declarationList = append(
			*l.astContext.declarationList,
			n,
		)
	case *parser.Call_inputContext:
		k := newIdentifier(c.Identifier().GetText(), true)
		l.astContext.callNode.Inputs[k] = &exprRPN{}
		l.astContext.exprNode = l.astContext.callNode.Inputs[k]
	}
}

func (l *wdlv1_1Listener) ExitEveryRule(ctx antlr.ParserRuleContext) {
	switch ctx.(type) {
	case *parser.DocumentContext:
		l.astContext.kindStack.pop()
	case *parser.Document_elementContext:
		l.astContext.declarationList = nil
	case *parser.Import_docContext:
		l.astContext.kindStack.pop()
		l.astContext.importNode = nil
	case *parser.WorkflowContext:
		l.astContext.kindStack.pop()
		l.astContext.workflowNode = nil
		l.astContext.declarationList = nil
	case *parser.Workflow_inputContext:
		l.astContext.kindStack.pop()
	case *parser.Workflow_outputContext:
		l.astContext.kindStack.pop()
	case *parser.Inner_workflow_elementContext:
		l.astContext.declarationList = nil
	case *parser.CallContext:
		l.astContext.kindStack.pop()
		l.astContext.callNode = nil
	case *parser.TaskContext:
		l.astContext.kindStack.pop()
		l.astContext.taskNode = nil
		l.astContext.declarationList = &l.wdl.Structs
	case *parser.Task_inputContext:
		l.astContext.kindStack.pop()
		l.astContext.declarationList = &l.astContext.taskNode.PrvtDecls
	case *parser.Task_outputContext:
		l.astContext.kindStack.pop()
		l.astContext.declarationList = &l.astContext.taskNode.PrvtDecls
	case *parser.Task_runtime_kvContext:
		l.astContext.exprNode = nil
	case *parser.MetaContext:
		l.astContext.kindStack.pop()
		l.astContext.metadataList = nil
	case *parser.Parameter_metaContext:
		l.astContext.kindStack.pop()
		l.astContext.metadataList = nil
	case *parser.Bound_declsContext:
		l.astContext.exprNode = nil
	case *parser.Call_inputContext:
		l.astContext.exprNode = nil
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
	n := newDecl(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.Identifier().GetText(),
		ctx.Wdl_type().GetText(),
	)
	*l.astContext.declarationList = append(
		*l.astContext.declarationList,
		n,
	)
}

// Parse metadata
func (l *wdlv1_1Listener) ExitMeta_kv(ctx *parser.Meta_kvContext) {
	k, v := ctx.MetaIdentifier().GetText(), ctx.Meta_value().GetText()
	(*l.astContext.metadataList)[k] = v
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
