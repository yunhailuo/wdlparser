package wdlparser

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"

	parser "github.com/yunhailuo/wdlparser/pkg/wdlv1_1"
)

type expr struct {
	genNode
	rpn  []interface{}
	lock sync.RWMutex
}

func (*expr) getKind() nodeKind { return exp }

func (e *expr) append(elem interface{}) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.rpn = append(e.rpn, elem)
}

func (e *expr) extend(e2 *expr) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e2.lock.Lock()
	defer e2.lock.Unlock()
	e.rpn = append(e.rpn, e2.rpn...)
}

func newExpr(start, end int) *expr {
	e := new(expr)
	e.genNode = genNode{start: start, end: end}
	return e
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

func newNone() value {
	return value{Any, nil}
}

// Operators

type wdlOpSym string

const (
	wdlNeg wdlOpSym = " -"
	wdlNot wdlOpSym = "!"
	wdlMul wdlOpSym = "*"
	wdlDiv wdlOpSym = "/"
	wdlMod wdlOpSym = "%"
	wdlAdd wdlOpSym = "+"
	wdlSub wdlOpSym = "-"
	wdlEq  wdlOpSym = "=="
	wdlNeq wdlOpSym = "!="
	wdlLt  wdlOpSym = "<"
	wdlLte wdlOpSym = "<="
	wdlGt  wdlOpSym = ">"
	wdlGte wdlOpSym = ">="
	wdlAnd wdlOpSym = "&&"
	wdlOr  wdlOpSym = "||"
)

var Operations = map[wdlOpSym]interface{}{
	wdlNeg: arithmeticNegation,
	wdlNot: logicalNegation,
	wdlMul: multiplication,
	wdlDiv: division,
	wdlMod: modulo,
	wdlAdd: addition,
	wdlSub: subtraction,
	wdlEq:  equality,
	wdlNeq: inequality,
	wdlLt:  less,
	wdlLte: lessEqual,
	wdlGt:  greater,
	wdlGte: greaterEqual,
	wdlAnd: logicalAnd,
	wdlOr:  logicalOr,
}

// Unary operators

func arithmeticNegation(rhs value) (value, error) {
	switch v := rhs.govalue.(type) {
	case float64:
		return value{Float, -v}, nil
	case int64:
		return value{Int, -v}, nil
	default:
		return newNone(), fmt.Errorf(
			"arithmetic negation doesn't support: %v of %T", v, v,
		)
	}
}

func logicalNegation(rhs value) (value, error) {
	switch v := rhs.govalue.(type) {
	case bool:
		return value{Boolean, !v}, nil
	default:
		return newNone(), fmt.Errorf(
			"logical negation doesn't support: %v of %T", v, v,
		)
	}
}

// Binary operators

func generalBinaryOp(lhs, rhs value, op wdlOpSym) (value, error) {
	typePair := [2]Type{lhs.typ, rhs.typ}
	switch {
	case typePair == [2]Type{Boolean, Boolean}:
		l := lhs.govalue.(bool)
		r := rhs.govalue.(bool)
		switch op {
		case wdlEq:
			return value{Boolean, l == r}, nil
		case wdlNeq:
			return value{Boolean, l != r}, nil
		case wdlLt:
			return value{
				Boolean, strconv.FormatBool(l) < strconv.FormatBool(r),
			}, nil
		case wdlLte:
			return value{
				Boolean, strconv.FormatBool(l) <= strconv.FormatBool(r),
			}, nil
		case wdlGt:
			return value{
				Boolean, strconv.FormatBool(l) > strconv.FormatBool(r),
			}, nil
		case wdlGte:
			return value{
				Boolean, strconv.FormatBool(l) >= strconv.FormatBool(r),
			}, nil
		case wdlAnd:
			return value{Boolean, l && r}, nil
		case wdlOr:
			return value{Boolean, l || r}, nil
		default:
			return newNone(), fmt.Errorf(
				"%v doesn't support: %v and %v", op, lhs.typ, rhs.typ,
			)
		}
	case typePair == [2]Type{Int, Int}:
		l := lhs.govalue.(int64)
		r := rhs.govalue.(int64)
		switch op {
		case wdlMul:
			return value{Int, l * r}, nil
		case wdlDiv:
			return value{Int, l / r}, nil
		case wdlMod:
			return value{Int, l % r}, nil
		case wdlAdd:
			return value{Int, l + r}, nil
		case wdlSub:
			return value{Int, l - r}, nil
		case wdlEq:
			return value{Boolean, l == r}, nil
		case wdlNeq:
			return value{Boolean, l != r}, nil
		case wdlLt:
			return value{Boolean, l < r}, nil
		case wdlLte:
			return value{Boolean, l <= r}, nil
		case wdlGt:
			return value{Boolean, l > r}, nil
		case wdlGte:
			return value{Boolean, l >= r}, nil
		default:
			return newNone(), fmt.Errorf(
				"%v doesn't support: %v and %v", op, lhs.typ, rhs.typ,
			)
		}
	case typePair == [2]Type{Float, Float}, typePair == [2]Type{Int, Float},
		typePair == [2]Type{Float, Int}:
		var l, r float64
		var lFloat, rFloat bool
		l, lFloat = lhs.govalue.(float64)
		if !lFloat {
			l = float64(lhs.govalue.(int64))
		}
		r, rFloat = rhs.govalue.(float64)
		if !rFloat {
			r = float64(rhs.govalue.(int64))
		}
		switch op {
		case wdlMul:
			return value{Float, l * r}, nil
		case wdlDiv:
			return value{Float, l / r}, nil
		case wdlMod:
			return value{Float, math.Mod(l, r)}, nil
		case wdlAdd:
			return value{Float, l + r}, nil
		case wdlSub:
			return value{Float, l - r}, nil
		case wdlEq:
			return value{Boolean, l == r}, nil
		case wdlNeq:
			return value{Boolean, l != r}, nil
		case wdlLt:
			return value{Boolean, l < r}, nil
		case wdlLte:
			return value{Boolean, l <= r}, nil
		case wdlGt:
			return value{Boolean, l > r}, nil
		case wdlGte:
			return value{Boolean, l >= r}, nil
		default:
			return newNone(), fmt.Errorf(
				"%v doesn't support: %v and %v", op, lhs.typ, rhs.typ,
			)
		}
	case typePair == [2]Type{String, String}:
		l := lhs.govalue.(string)
		r := rhs.govalue.(string)
		switch op {
		case wdlAdd:
			return value{String, l + r}, nil
		case wdlEq:
			return value{Boolean, l == r}, nil
		case wdlNeq:
			return value{Boolean, l != r}, nil
		case wdlLt:
			return value{Boolean, l < r}, nil
		case wdlLte:
			return value{Boolean, l <= r}, nil
		case wdlGt:
			return value{Boolean, l > r}, nil
		case wdlGte:
			return value{Boolean, l >= r}, nil
		default:
			return newNone(), fmt.Errorf(
				"%v doesn't support: %v and %v", op, lhs.typ, rhs.typ,
			)
		}
	case typePair == [2]Type{String, Int}, typePair == [2]Type{Int, String},
		typePair == [2]Type{String, Float}, typePair == [2]Type{Float, String}:
		var l, r string
		switch lhs.typ {
		case Int:
			l = strconv.FormatInt(lhs.govalue.(int64), 10)
		case Float:
			l = strconv.FormatFloat(lhs.govalue.(float64), 'G', -1, 64)
		case String:
			l = lhs.govalue.(string)
		}
		switch rhs.typ {
		case Int:
			r = strconv.FormatInt(rhs.govalue.(int64), 10)
		case Float:
			r = strconv.FormatFloat(rhs.govalue.(float64), 'G', -1, 64)
		case String:
			r = rhs.govalue.(string)
		}
		switch op {
		case wdlAdd:
			return value{String, l + r}, nil
		default:
			return newNone(), fmt.Errorf(
				"%v doesn't support: %v and %v", op, lhs.typ, rhs.typ,
			)
		}
	case typePair == [2]Type{String, File}, typePair == [2]Type{File, String},
		typePair == [2]Type{File, File}:
		l := lhs.govalue.(string)
		r := rhs.govalue.(string)
		switch op {
		case wdlAdd:
			return value{File, l + r}, nil
		case wdlEq:
			return value{Boolean, l == r}, nil
		case wdlNeq:
			return value{Boolean, l != r}, nil
		default:
			return newNone(), fmt.Errorf(
				"%v doesn't support: %v and %v", op, lhs.typ, rhs.typ,
			)
		}
	default:
		return newNone(), fmt.Errorf(
			"operation %v doesn't support: %v and %v in generalBinaryOp",
			op, lhs.typ, rhs.typ,
		)
	}
}

func multiplication(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlMul)
}

func division(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlDiv)
}

func modulo(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlMod)
}
func addition(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlAdd)
}

func subtraction(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlSub)
}

func equality(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlEq)
}
func inequality(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlNeq)
}
func less(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlLt)
}
func lessEqual(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlLte)
}
func greater(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlGt)
}
func greaterEqual(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlGte)
}
func logicalAnd(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlAnd)
}
func logicalOr(lhs, rhs value) (value, error) {
	return generalBinaryOp(lhs, rhs, wdlOr)
}

// Antlr4 listeners

func (l *wdlv1_1Listener) EnterExpr(ctx *parser.ExprContext) {
	l.branching(
		newExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitExpr(ctx *parser.ExprContext) {
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) ExitPrimitive_literal(
	ctx *parser.Primitive_literalContext,
) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"number",
				"expression",
				l.currentNode,
			),
		)
	}

	// BoolLiteral of primitive_literal
	boolToken := ctx.BoolLiteral()
	if boolToken != nil {
		v, e := newValue(Boolean, boolToken.GetText())
		if e == nil {
			expr.append(v)
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
			expr.append(v)
		} else {
			log.Fatal(e)
		}
		return
	}

	// Identifier of primitive_literal
	// TODO: this should somehow point to the variable
	identifierToken := ctx.Identifier()
	if identifierToken != nil {
		expr.append(identifierToken)
		return
	}
}

func (l *wdlv1_1Listener) ExitNumber(ctx *parser.NumberContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"number",
				"expression",
				l.currentNode,
			),
		)
	}

	// IntLiteral
	intToken := ctx.IntLiteral()
	if intToken != nil {
		v, e := newValue(Int, intToken.GetText())
		if e == nil {
			expr.append(v)
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
			expr.append(v)
		} else {
			log.Fatal(e)
		}
		return
	}

	log.Fatalf("Failed to parse %v: %v", "Number", ctx.GetText())
}

func (l *wdlv1_1Listener) ExitLor(ctx *parser.LorContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"logical OR",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlOr)
}

func (l *wdlv1_1Listener) ExitLand(ctx *parser.LandContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"logical AND",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlAnd)
}

func (l *wdlv1_1Listener) ExitEqeq(ctx *parser.EqeqContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"equality",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlEq)
}

func (l *wdlv1_1Listener) ExitNeq(ctx *parser.NeqContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"inequality",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlNeq)
}

func (l *wdlv1_1Listener) ExitLte(ctx *parser.LteContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"less than or equal",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlLte)
}

func (l *wdlv1_1Listener) ExitGte(ctx *parser.GteContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"grater than or equal",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlGte)
}

func (l *wdlv1_1Listener) ExitLt(ctx *parser.LtContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"less than",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlLt)
}

func (l *wdlv1_1Listener) ExitGt(ctx *parser.GtContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"greater than",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlGt)
}

func (l *wdlv1_1Listener) ExitAdd(ctx *parser.AddContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"addition",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlAdd)
}

func (l *wdlv1_1Listener) ExitSub(ctx *parser.SubContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"substract",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlSub)
}

func (l *wdlv1_1Listener) ExitMul(ctx *parser.MulContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"multiplication",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlMul)
}

func (l *wdlv1_1Listener) ExitDivide(ctx *parser.DivideContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"division",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlDiv)
}

func (l *wdlv1_1Listener) ExitMod(ctx *parser.ModContext) {
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"remainder",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.append(wdlMod)
}

func (l *wdlv1_1Listener) EnterExpression_group(
	ctx *parser.Expression_groupContext,
) {
	l.branching(
		newGenNode(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitExpression_group(
	ctx *parser.Expression_groupContext,
) {
	var childExprs []*expr
	for _, child := range l.currentNode.getChildren() {
		if ce, isExpr := child.(*expr); isExpr {
			childExprs = append(childExprs, ce)
		}
	}
	if len(childExprs) != 1 {
		log.Fatalf(
			"expression group expects exactly 1 child expression, found %v",
			len(childExprs),
		)
	}
	l.currentNode = l.currentNode.getParent()
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"grouping",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.extend(childExprs[0])
}

func (l *wdlv1_1Listener) EnterNegate(ctx *parser.NegateContext) {
	l.branching(
		newGenNode(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitNegate(ctx *parser.NegateContext) {
	var childExprs []*expr
	for _, child := range l.currentNode.getChildren() {
		if ce, isExpr := child.(*expr); isExpr {
			childExprs = append(childExprs, ce)
		}
	}
	if len(childExprs) != 1 {
		log.Fatalf(
			"logical not expects exactly 1 child expression, found %v",
			len(childExprs),
		)
	}
	l.currentNode = l.currentNode.getParent()
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"logical NOT",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.extend(childExprs[0])
	expr.append(wdlNot)
}

func (l *wdlv1_1Listener) EnterUnarysigned(ctx *parser.UnarysignedContext) {
	l.branching(
		newGenNode(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitUnarysigned(ctx *parser.UnarysignedContext) {
	var childExprs []*expr
	for _, child := range l.currentNode.getChildren() {
		if ce, isExpr := child.(*expr); isExpr {
			childExprs = append(childExprs, ce)
		}
	}
	if len(childExprs) != 1 {
		log.Fatalf(
			"arithmetic negation expects exactly 1 child expression, found %v",
			len(childExprs),
		)
	}
	l.currentNode = l.currentNode.getParent()
	expr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"unary negation",
				"expression",
				l.currentNode,
			),
		)
	}
	expr.extend(childExprs[0])
	if ctx.MINUS() != nil {
		expr.append(wdlNeg)
	}
}
