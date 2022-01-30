package wdlparser

import (
	"fmt"
	"log"
	"strconv"

	parser "github.com/yunhailuo/wdlparser/pkg/antlr4_grammar/1_1"
)

// Reverse Polish notation of expression
type exprRPN []interface{}

func (e *exprRPN) append(elem interface{}) {
	*e = append(*e, elem)
}

type expression struct {
	genNode
	rpn      exprRPN
	subExprs exprStack
}

func newExpression(start, end int) *expression {
	return &expression{
		genNode: genNode{start: start, end: end},
	}
}

type exprStack []*expression

func (s *exprStack) push(e *expression) {
	*s = append(*s, e)
}

func (s *exprStack) pop() *expression {
	stackDepth := len(*s)
	if stackDepth > 0 {
		e := (*s)[stackDepth-1]
		// Won't zero the popped element since nodeKind is limited and small
		*s = (*s)[:stackDepth-1]
		return e
	}
	log.Fatalf("pop error: expression stack %v is empty", *s)
	return nil
}

// A Type represents a type of WDL.
// All types implement the Type interface.
type Type interface {
	typeString() string
}

type primitive string

func (p primitive) typeString() string { return string(p) }

const (
	Boolean = primitive("Boolean")
	Int     = primitive("Int")
	Float   = primitive("Float")
	String  = primitive("String")
	File    = primitive("File")
	Any     = primitive("Any")
)

// A value represents a value in WDL.
type value struct {
	typ     Type
	govalue interface{} // actual underlying go value
}

func newValue(typ Type, raw string) (value, error) {
	v := new(value)
	v.typ = typ
	var e error = nil
	switch typ {
	case Boolean:
		v.govalue, e = strconv.ParseBool(raw)
	case Int:
		v.govalue, e = strconv.ParseInt(raw, 10, 64)
	case Float:
		v.govalue, e = strconv.ParseFloat(raw, 64)
	case String, File:
		v.govalue = raw
	case Any:
		v.govalue = nil
	default:
		v.govalue = nil
		e = fmt.Errorf("unsupported primitive: %v", typ)
	}
	return *v, e
}

// Operators

type WDLOpSym string

const (
	WDLNeg WDLOpSym = "^-"
	WDLNot WDLOpSym = "!"
	WDLStr WDLOpSym = "str"
	WDLMul WDLOpSym = "*"
	WDLDiv WDLOpSym = "/"
	WDLMod WDLOpSym = "%"
	WDLAdd WDLOpSym = "+"
	WDLSub WDLOpSym = "-"
	WDLEq  WDLOpSym = "=="
	WDLNeq WDLOpSym = "!="
	WDLLt  WDLOpSym = "<"
	WDLLte WDLOpSym = "<="
	WDLGt  WDLOpSym = ">"
	WDLGte WDLOpSym = ">="
	WDLAnd WDLOpSym = "&&"
	WDLOr  WDLOpSym = "||"
)

// Antlr4 listeners

func (l *wdlv1_1Listener) EnterExpr(ctx *parser.ExprContext) {
	e := newExpression(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
	)
	e.setParent(l.astContext.exprNode)
	l.astContext.exprNode.subExprs.push(e)
	l.astContext.exprNode = e
}

func (l *wdlv1_1Listener) ExitExpr(ctx *parser.ExprContext) {
	l.astContext.exprNode = l.astContext.exprNode.getParent().(*expression)
}

func (l *wdlv1_1Listener) ExitPrimitive_literal(
	ctx *parser.Primitive_literalContext,
) {
	// BoolLiteral of primitive_literal
	boolToken := ctx.BoolLiteral()
	if boolToken != nil {
		v, e := newValue(Boolean, boolToken.GetText())
		if e == nil {
			l.astContext.exprNode.rpn.append(v)
		} else {
			log.Fatal(e)
		}
		return
	}

	// NONELITERAL of primitive_literal
	noneToken := ctx.NONELITERAL()
	if noneToken != nil {
		v, e := newValue(Any, noneToken.GetText())
		if e == nil {
			l.astContext.exprNode.rpn.append(v)
		} else {
			log.Fatal(e)
		}
		return
	}

	// Identifier of primitive_literal
	// TODO: this should somehow point to the variable
	identifierToken := ctx.Identifier()
	if identifierToken != nil {
		l.astContext.exprNode.rpn.append(
			newIdentifier(identifierToken.GetText(), true),
		)
		return
	}
}

func (l *wdlv1_1Listener) ExitNumber(ctx *parser.NumberContext) {
	// IntLiteral
	intToken := ctx.IntLiteral()
	if intToken != nil {
		v, e := newValue(Int, intToken.GetText())
		if e == nil {
			l.astContext.exprNode.rpn.append(v)
		} else {
			log.Fatal(e)
		}
		return
	}

	// FloatLiteral
	floatToken := ctx.FloatLiteral()
	if floatToken != nil {
		v, e := newValue(Float, floatToken.GetText())
		if e == nil {
			l.astContext.exprNode.rpn.append(v)
		} else {
			log.Fatal(e)
		}
		return
	}

	log.Fatalf("Failed to parse %v: %v", "Number", ctx.GetText())
}

func (l *wdlv1_1Listener) ExitString_part(ctx *parser.String_partContext) {
	v, e := newValue(String, ctx.GetText())
	if e == nil {
		l.astContext.exprNode.rpn.append(v)
	} else {
		log.Fatal(e)
	}
}
func (l *wdlv1_1Listener) ExitString_expr_part(
	ctx *parser.String_expr_partContext,
) {
	e := l.astContext.exprNode.subExprs.pop()
	l.astContext.exprNode.rpn.append(e)
	l.astContext.exprNode.rpn.append(WDLStr)
}

func (l *wdlv1_1Listener) ExitString_expr_with_string_part(
	ctx *parser.String_expr_with_string_partContext,
) {
	// join expr and string within string_expr_with_string_part
	l.astContext.exprNode.rpn.append(WDLAdd)
	// join others in wdl_string
	l.astContext.exprNode.rpn.append(WDLAdd)
}

func (l *wdlv1_1Listener) ExitLor(ctx *parser.LorContext) {
	l.astContext.exprNode.rpn.append(WDLOr)
}

func (l *wdlv1_1Listener) ExitLand(ctx *parser.LandContext) {
	l.astContext.exprNode.rpn.append(WDLAnd)
}

func (l *wdlv1_1Listener) ExitEqeq(ctx *parser.EqeqContext) {
	l.astContext.exprNode.rpn.append(WDLEq)
}

func (l *wdlv1_1Listener) ExitNeq(ctx *parser.NeqContext) {
	l.astContext.exprNode.rpn.append(WDLNeq)
}

func (l *wdlv1_1Listener) ExitLte(ctx *parser.LteContext) {
	l.astContext.exprNode.rpn.append(WDLLte)
}

func (l *wdlv1_1Listener) ExitGte(ctx *parser.GteContext) {
	l.astContext.exprNode.rpn.append(WDLGte)
}

func (l *wdlv1_1Listener) ExitLt(ctx *parser.LtContext) {
	l.astContext.exprNode.rpn.append(WDLLt)
}

func (l *wdlv1_1Listener) ExitGt(ctx *parser.GtContext) {
	l.astContext.exprNode.rpn.append(WDLGt)
}

func (l *wdlv1_1Listener) ExitAdd(ctx *parser.AddContext) {
	l.astContext.exprNode.rpn.append(WDLAdd)
}

func (l *wdlv1_1Listener) ExitSub(ctx *parser.SubContext) {
	l.astContext.exprNode.rpn.append(WDLSub)
}

func (l *wdlv1_1Listener) ExitMul(ctx *parser.MulContext) {
	l.astContext.exprNode.rpn.append(WDLMul)
}

func (l *wdlv1_1Listener) ExitDivide(ctx *parser.DivideContext) {
	l.astContext.exprNode.rpn.append(WDLDiv)
}

func (l *wdlv1_1Listener) ExitMod(ctx *parser.ModContext) {
	l.astContext.exprNode.rpn.append(WDLMod)
}

func (l *wdlv1_1Listener) ExitNegate(ctx *parser.NegateContext) {
	e := l.astContext.exprNode.subExprs.pop()
	l.astContext.exprNode.rpn.append(e)
	l.astContext.exprNode.rpn.append(WDLNot)
}

func (l *wdlv1_1Listener) ExitExpression_group(
	ctx *parser.Expression_groupContext,
) {
	e := l.astContext.exprNode.subExprs.pop()
	l.astContext.exprNode.rpn.append(e)
}

func (l *wdlv1_1Listener) ExitUnarysigned(ctx *parser.UnarysignedContext) {
	e := l.astContext.exprNode.subExprs.pop()
	l.astContext.exprNode.rpn.append(e)
	if ctx.MINUS() != nil {
		l.astContext.exprNode.rpn.append(WDLNeg)
	}
}
