package wdlparser

import (
	"fmt"
	"log"
	"math"
	"strconv"

	parser "github.com/yunhailuo/wdlparser/pkg/wdlv1_1"
)

type Type interface {
	typeString() string
}

type primitive string

func (p primitive) typeString() string { return string(p) }

const Boolean = primitive("Boolean")
const Int = primitive("Int")
const Float = primitive("Float")
const String = primitive("String")
const Any = primitive("Any")

type value struct {
	typ      Type
	govalue  interface{} // actual underlying go value
	nullable bool
	nonEmpty bool // array only
}

func makeNone() value {
	return value{Any, nil, false, false}
}

func primitiveFromLiteral(lit string, typ Type) (value, error) {
	v := new(value)
	v.typ = typ
	var e error = nil
	pT, isPrimitive := typ.(primitive)
	if !isPrimitive {
		e = fmt.Errorf("only support primitive type, got typ: %T", typ)
		return makeNone(), e
	}
	switch pT {
	case Boolean:
		v.govalue, e = strconv.ParseBool(lit)
	case Int:
		v.govalue, e = strconv.ParseInt(lit, 10, 64)
	case Float:
		v.govalue, e = strconv.ParseFloat(lit, 64)
	case String:
		v.govalue = lit
	case Any:
		v.govalue = nil
	default:
		v.govalue = nil
		e = fmt.Errorf("unsupported %T primitive: %v", pT, pT)
	}
	return *v, e
}

type evaluator func() (value, error)

type expr struct {
	vertex
	x     *expr // left operand
	y     *expr // right operand
	opSym string
	eval  evaluator // evaluate this expression (such as adding [x] to [y])
}

func newExpr(start, end int, opSym string) *expr {
	e := new(expr)
	e.vertex = vertex{start: start, end: end, kind: exp}
	e.opSym = opSym
	e.eval = func() (value, error) {
		log.Fatalf(
			"\"eval\" function undefined for expression \"%v\" at %d:%d",
			opSym, start, end,
		)
		return makeNone(), nil
	}
	return e
}

func (e *expr) getChildExprs() []*expr {
	var exprs []*expr
	for _, child := range e.getChildren() {
		if ce, isExpr := child.(*expr); isExpr {
			exprs = append(exprs, ce)
		}
	}
	return exprs
}

func (l *wdlv1_1Listener) ExitPrimitive_literal(
	ctx *parser.Primitive_literalContext,
) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		"primitive literal",
	)

	// BoolLiteral of primitive_literal
	boolToken := ctx.BoolLiteral()
	if boolToken != nil {
		e.eval = func() (value, error) {
			return primitiveFromLiteral(boolToken.GetText(), Boolean)
		}
		l.branching(e, false)
	}
}

func (l *wdlv1_1Listener) ExitNumber(ctx *parser.NumberContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		"",
	)

	// IntLiteral
	intToken := ctx.IntLiteral()
	if intToken != nil {
		e.eval = func() (value, error) {
			return primitiveFromLiteral(intToken.GetText(), Int)
		}
		l.branching(e, false)
	}
	// FloatLiteral
	floatToken := ctx.FloatLiteral()
	if floatToken != nil {
		e.eval = func() (value, error) {
			return primitiveFromLiteral(floatToken.GetText(), Float)
		}
		l.branching(e, false)
	}
}

func wdlOr(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for OR operation", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	xIsBool, yIsBool := x.typ == Boolean, y.typ == Boolean
	switch {
	case xIsBool && yIsBool:
		return value{
			typ: Boolean, govalue: x.govalue.(bool) || y.govalue.(bool),
		}, nil
	case xIsBool && (!yIsBool): // only y is invalid
		return makeNone(), fmt.Errorf(
			"found right operand is not bool, " +
				"expect both operands being bool for OR operation",
		)
	case (!xIsBool) && yIsBool: // only x is invalid
		return makeNone(), fmt.Errorf(
			"found left operand is not bool, " +
				"expect both operands being bool for OR operation",
		)
	default: // neither operands is valid
		return makeNone(), fmt.Errorf(
			"found neither left or right operands is bool, " +
				"expect both operands being bool for OR operation",
		)
	}
}

func (l *wdlv1_1Listener) EnterLor(ctx *parser.LorContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.OR().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlOr(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitLor(ctx *parser.LorContext) {
	lorExpr, isExpr := l.currentNode.(*expr)
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
	childExprs := lorExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Logical OR expression expect 2 expressions as operand, found %v",
			operandCount,
		)
	}
	lorExpr.x, lorExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func wdlAnd(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for AND operation", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	xIsBool, yIsBool := x.typ == Boolean, y.typ == Boolean
	switch {
	case xIsBool && yIsBool:
		return value{
			typ: Boolean, govalue: x.govalue.(bool) && y.govalue.(bool),
		}, nil
	case xIsBool && (!yIsBool): // only y is invalid
		return makeNone(), fmt.Errorf(
			"found right operand is not bool, " +
				"expect both operands being bool for AND operation",
		)
	case (!xIsBool) && yIsBool: // only x is invalid
		return makeNone(), fmt.Errorf(
			"found left operand is not bool, " +
				"expect both operands being bool for AND operation",
		)
	default: // neither operands is valid
		return makeNone(), fmt.Errorf(
			"found neither left or right operands is bool, " +
				"expect both operands being bool for AND operation",
		)
	}
}

func (l *wdlv1_1Listener) EnterLand(ctx *parser.LandContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.AND().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlAnd(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitLand(ctx *parser.LandContext) {
	lorExpr, isExpr := l.currentNode.(*expr)
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
	childExprs := lorExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Logical AND expression expect 2 expressions as operand, found %v",
			operandCount,
		)
	}
	lorExpr.x, lorExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func wdlNegate(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 1 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 1 for negation", operandCount,
		)
	}
	x := operands[0]
	switch x.typ {
	case Boolean:
		return value{typ: Boolean, govalue: !x.govalue.(bool)}, nil
	case Int:
		return value{typ: Int, govalue: -x.govalue.(int64)}, nil
	case Float:
		return value{typ: Float, govalue: -x.govalue.(float64)}, nil
	default:
		return makeNone(), fmt.Errorf(
			"invalid operand: negation is only valid for boolean, int or float",
		)
	}
}

func (l *wdlv1_1Listener) EnterNegate(ctx *parser.NegateContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.NOT().GetText(),
	)
	e.eval = func() (value, error) {
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlNegate(y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitNegate(ctx *parser.NegateContext) {
	negateExpr, isExpr := l.currentNode.(*expr)
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
	childExprs := negateExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 1 {
		log.Fatalf(
			"Logical NOT expression expect 1 expression as operand, found %v",
			operandCount,
		)
	}
	negateExpr.y = childExprs[0]
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterUnarysigned(ctx *parser.UnarysignedContext) {
	var opSym string = "+"
	if ctx.MINUS() != nil {
		opSym = "-"
	}
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		opSym,
	)
	e.eval = func() (value, error) {
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		switch opSym {
		case "-":
			return wdlNegate(y)
		default: // +:
			return y, nil // potential risk: y type is not checked
		}
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitUnarysigned(ctx *parser.UnarysignedContext) {
	unaryExpr, isExpr := l.currentNode.(*expr)
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
	childExprs := unaryExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 1 {
		log.Fatalf(
			"Unary +/- expression expect 1 expressions as operand, found %v",
			operandCount,
		)
	}
	unaryExpr.y = childExprs[0]
	l.currentNode = l.currentNode.getParent()
}

func wdlLt(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for less than comparison",
			operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == Int && y.typ == Int:
		return value{
			typ: Boolean, govalue: (x.govalue.(int64)) < (y.govalue.(int64)),
		}, nil
	case x.typ == Float && y.typ == Int:
		return value{
			typ:     Boolean,
			govalue: (x.govalue.(float64)) < float64(y.govalue.(int64)),
		}, nil
	case x.typ == Int && y.typ == Float:
		return value{
			typ:     Boolean,
			govalue: float64(x.govalue.(int64)) < (y.govalue.(float64)),
		}, nil
	case x.typ == Float && y.typ == Float:
		return value{
			typ:     Boolean,
			govalue: (x.govalue.(float64)) < (y.govalue.(float64)),
		}, nil
	case x.typ == String && y.typ == String:
		return value{
			typ: Boolean, govalue: (x.govalue.(string)) < (y.govalue.(string)),
		}, nil
	default: // neither operands is int, float or string
		return makeNone(), fmt.Errorf(
			"invalid operands: less than comparison is only valid for" +
				" int, float or string",
		)
	}
}

func (l *wdlv1_1Listener) EnterLt(ctx *parser.LtContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.LT().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlLt(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitLt(ctx *parser.LtContext) {
	ltExpr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"less than comparison",
				"expression",
				l.currentNode,
			),
		)
	}
	childExprs := ltExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Less than comparison expect 2 expressions as operand, found %v",
			operandCount,
		)
	}
	ltExpr.x, ltExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func wdlLe(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for less than or equal to comparison",
			operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == Int && y.typ == Int:
		return value{
			typ: Boolean, govalue: (x.govalue.(int64)) <= (y.govalue.(int64)),
		}, nil
	case x.typ == Float && y.typ == Int:
		return value{
			typ:     Boolean,
			govalue: (x.govalue.(float64)) <= float64(y.govalue.(int64)),
		}, nil
	case x.typ == Int && y.typ == Float:
		return value{
			typ:     Boolean,
			govalue: float64(x.govalue.(int64)) <= (y.govalue.(float64)),
		}, nil
	case x.typ == Float && y.typ == Float:
		return value{
			typ:     Boolean,
			govalue: (x.govalue.(float64)) <= (y.govalue.(float64)),
		}, nil
	case x.typ == String && y.typ == String:
		return value{
			typ: Boolean, govalue: (x.govalue.(string)) <= (y.govalue.(string)),
		}, nil
	default: // neither operands is int, float or string
		return makeNone(), fmt.Errorf(
			"invalid operands: less than or equal to comparison is only valid" +
				" for int, float or string",
		)
	}
}

func (l *wdlv1_1Listener) EnterLte(ctx *parser.LteContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.LTE().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlLe(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitLte(ctx *parser.LteContext) {
	lteExpr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"less than or equal to comparison",
				"expression",
				l.currentNode,
			),
		)
	}
	childExprs := lteExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Less than or equal to comparison expect 2 expressions as operand,"+
				" found %v",
			operandCount,
		)
	}
	lteExpr.x, lteExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func wdlGe(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for greater than or equal to"+
				" comparison",
			operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == Int && y.typ == Int:
		return value{
			typ: Boolean, govalue: (x.govalue.(int64)) >= (y.govalue.(int64)),
		}, nil
	case x.typ == Float && y.typ == Int:
		return value{
			typ:     Boolean,
			govalue: (x.govalue.(float64)) >= float64(y.govalue.(int64)),
		}, nil
	case x.typ == Int && y.typ == Float:
		return value{
			typ:     Boolean,
			govalue: float64(x.govalue.(int64)) >= (y.govalue.(float64)),
		}, nil
	case x.typ == Float && y.typ == Float:
		return value{
			typ:     Boolean,
			govalue: (x.govalue.(float64)) >= (y.govalue.(float64)),
		}, nil
	case x.typ == String && y.typ == String:
		return value{
			typ: Boolean, govalue: (x.govalue.(string)) >= (y.govalue.(string)),
		}, nil
	default: // neither operands is int, float or string
		return makeNone(), fmt.Errorf(
			"invalid operands: greater than or equal to comparison is" +
				" only valid for int, float or string",
		)
	}
}

func (l *wdlv1_1Listener) EnterGte(ctx *parser.GteContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.GTE().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlGe(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitGte(ctx *parser.GteContext) {
	gteExpr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"greater than or equal to comparison",
				"expression",
				l.currentNode,
			),
		)
	}
	childExprs := gteExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Greater than or equal to comparison expect 2 expressions"+
				" as operand, found %v",
			operandCount,
		)
	}
	gteExpr.x, gteExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func wdlGt(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for greater than comparison",
			operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == Int && y.typ == Int:
		return value{
			typ: Boolean, govalue: (x.govalue.(int64)) > (y.govalue.(int64)),
		}, nil
	case x.typ == Float && y.typ == Int:
		return value{
			typ:     Boolean,
			govalue: (x.govalue.(float64)) > float64(y.govalue.(int64)),
		}, nil
	case x.typ == Int && y.typ == Float:
		return value{
			typ:     Boolean,
			govalue: float64(x.govalue.(int64)) > (y.govalue.(float64)),
		}, nil
	case x.typ == Float && y.typ == Float:
		return value{
			typ:     Boolean,
			govalue: (x.govalue.(float64)) > (y.govalue.(float64)),
		}, nil
	case x.typ == String && y.typ == String:
		return value{
			typ: Boolean, govalue: (x.govalue.(string)) > (y.govalue.(string)),
		}, nil
	default: // neither operands is int, float or string
		return makeNone(), fmt.Errorf(
			"invalid operands: greater than comparison is only valid" +
				" for int, float or string",
		)
	}
}

func (l *wdlv1_1Listener) EnterGt(ctx *parser.GtContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.GT().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlGt(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitGt(ctx *parser.GtContext) {
	gtExpr, isExpr := l.currentNode.(*expr)
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
	childExprs := gtExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Greater than comparison expect 2 expressions as operand, found %v",
			operandCount,
		)
	}
	gtExpr.x, gtExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func wdlMul(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for multiplication", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == Int && y.typ == Int:
		return value{
			typ: Int, govalue: (x.govalue.(int64)) * (y.govalue.(int64)),
		}, nil
	case x.typ == Float && y.typ == Int:
		return value{
			typ:     Float,
			govalue: (x.govalue.(float64)) * float64(y.govalue.(int64)),
		}, nil
	case x.typ == Int && y.typ == Float:
		return value{
			typ:     Float,
			govalue: float64(x.govalue.(int64)) * (y.govalue.(float64)),
		}, nil
	case x.typ == Float && y.typ == Float:
		return value{
			typ: Float, govalue: (x.govalue.(float64)) * (y.govalue.(float64)),
		}, nil
	default: // neither operands is int or float
		return makeNone(), fmt.Errorf(
			"invalid operands: multiplication is only valid for int or float",
		)
	}
}

func (l *wdlv1_1Listener) EnterMul(ctx *parser.MulContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.STAR().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlMul(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitMul(ctx *parser.MulContext) {
	mulExpr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"multiply",
				"expression",
				l.currentNode,
			),
		)
	}
	childExprs := mulExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Multiply expression expect 2 expressions as operand, found %v",
			operandCount,
		)
	}
	mulExpr.x, mulExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func wdlDiv(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for division", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == Int && y.typ == Int:
		return value{
			typ: Int, govalue: (x.govalue.(int64)) / (y.govalue.(int64)),
		}, nil
	case x.typ == Float && y.typ == Int:
		return value{
			typ:     Float,
			govalue: (x.govalue.(float64)) / float64(y.govalue.(int64)),
		}, nil
	case x.typ == Int && y.typ == Float:
		return value{
			typ:     Float,
			govalue: float64(x.govalue.(int64)) / (y.govalue.(float64)),
		}, nil
	case x.typ == Float && y.typ == Float:
		return value{
			typ: Float, govalue: (x.govalue.(float64)) / (y.govalue.(float64)),
		}, nil
	default: // neither operands is int or float
		return makeNone(), fmt.Errorf(
			"invalid operands: division is only valid for int or float",
		)
	}
}

func (l *wdlv1_1Listener) EnterDivide(ctx *parser.DivideContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.DIVIDE().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlDiv(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitDivide(ctx *parser.DivideContext) {
	divideExpr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"divide",
				"expression",
				l.currentNode,
			),
		)
	}
	childExprs := divideExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Divide expression expect 2 expressions as operand, found %v",
			operandCount,
		)
	}
	divideExpr.x, divideExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func wdlMod(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for modulo", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == Int && y.typ == Int:
		return value{
			typ: Int, govalue: (x.govalue.(int64)) % (y.govalue.(int64)),
		}, nil
	case x.typ == Float && y.typ == Int:
		return value{
			typ:     Float,
			govalue: math.Mod(x.govalue.(float64), float64(y.govalue.(int64))),
		}, nil
	case x.typ == Int && y.typ == Float:
		return value{
			typ:     Float,
			govalue: math.Mod(float64(x.govalue.(int64)), y.govalue.(float64)),
		}, nil
	case x.typ == Float && y.typ == Float:
		return value{
			typ:     Float,
			govalue: math.Mod(x.govalue.(float64), y.govalue.(float64)),
		}, nil
	default: // neither operands is int or float
		return makeNone(), fmt.Errorf(
			"invalid operands: modulo is only valid for int or float",
		)
	}
}

func (l *wdlv1_1Listener) EnterMod(ctx *parser.ModContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.MOD().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlMod(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitMod(ctx *parser.ModContext) {
	modExpr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"modulo",
				"expression",
				l.currentNode,
			),
		)
	}
	childExprs := modExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Modulo expression expect 2 expressions as operand, found %v",
			operandCount,
		)
	}
	modExpr.x, modExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func wdlSub(operands ...value) (value, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return makeNone(), fmt.Errorf(
			"found %d operands, expect 2 for subtraction", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == Int && y.typ == Int:
		return value{
			typ: Int, govalue: (x.govalue.(int64)) - (y.govalue.(int64)),
		}, nil
	case x.typ == Float && y.typ == Int:
		return value{
			typ:     Float,
			govalue: (x.govalue.(float64)) - float64(y.govalue.(int64)),
		}, nil
	case x.typ == Int && y.typ == Float:
		return value{
			typ:     Float,
			govalue: float64(x.govalue.(int64)) - (y.govalue.(float64)),
		}, nil
	case x.typ == Float && y.typ == Float:
		return value{
			typ: Float, govalue: (x.govalue.(float64)) - (y.govalue.(float64)),
		}, nil
	default: // neither operands is int or float
		return makeNone(), fmt.Errorf(
			"invalid operands: subtraction is only valid for int or float",
		)
	}
}

func (l *wdlv1_1Listener) EnterSub(ctx *parser.SubContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.MINUS().GetText(),
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return makeNone(), errY
		}
		return wdlSub(x, y)
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitSub(ctx *parser.SubContext) {
	subExpr, isExpr := l.currentNode.(*expr)
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
	childExprs := subExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 2 {
		log.Fatalf(
			"Substract expression expect 2 expressions as operand, found %v",
			operandCount,
		)
	}
	subExpr.x, subExpr.y = childExprs[0], childExprs[1]
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterExpression_group(
	ctx *parser.Expression_groupContext,
) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		"()",
	)
	e.eval = func() (value, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return makeNone(), errX
		}
		return x, nil
	}
	l.branching(e, true)
}

func (l *wdlv1_1Listener) ExitExpression_group(
	ctx *parser.Expression_groupContext,
) {
	groupExpr, isExpr := l.currentNode.(*expr)
	if !isExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"expression group",
				"expression",
				l.currentNode,
			),
		)
	}
	childExprs := groupExpr.getChildExprs()
	operandCount := len(childExprs)
	if operandCount != 1 {
		log.Fatalf(
			"Expression group expect 1 expression as operand, found %v",
			operandCount,
		)
	}
	groupExpr.x = childExprs[0]
	l.currentNode = l.currentNode.getParent()
}
