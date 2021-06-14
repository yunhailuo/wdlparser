package wdlparser

import (
	"fmt"
	"log"
	"math"
	"strconv"

	parser "github.com/yunhailuo/wdlparser/pkg/wdlv1_1"
)

// A Type represents a type of WDL.
// All types implement the Type interface.
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

// A value represents a value in WDL.
type value struct {
	typ     Type
	govalue interface{} // actual underlying go value
}

func makeNone() value {
	return value{Any, nil}
}

type evaluator interface {
	node
	eval() (value, error)
}

type primitiveLiteral struct {
	genNode
	literal string
	typ     primitive
}

func newPrimitiveLiteral(
	start, end int, lit string, typ primitive,
) *primitiveLiteral {
	p := new(primitiveLiteral)
	p.genNode = genNode{start: start, end: end}
	p.literal = lit
	p.typ = typ
	return p
}

func (*primitiveLiteral) getKind() nodeKind { return exp }

func (p primitiveLiteral) eval() (value, error) {
	v := new(value)
	v.typ = p.typ
	var e error = nil
	switch p.typ {
	case Boolean:
		v.govalue, e = strconv.ParseBool(p.literal)
	case Int:
		v.govalue, e = strconv.ParseInt(p.literal, 10, 64)
	case Float:
		v.govalue, e = strconv.ParseFloat(p.literal, 64)
	case String:
		v.govalue = p.literal
	case Any:
		v.govalue = nil
	default:
		v.govalue = nil
		e = fmt.Errorf("unsupported primitive: %v", p.typ)
	}
	return *v, e
}

// A unaryExpr node represents a unary operation.
type unaryExpr struct {
	genNode
	x     evaluator // operand
	opSym string
}

func newUnaryExpr(start, end int, symbol string) *unaryExpr {
	u := new(unaryExpr)
	u.genNode = genNode{start: start, end: end}
	u.opSym = symbol
	return u
}

func (*unaryExpr) getKind() nodeKind { return exp }

func (u unaryExpr) eval() (value, error) {
	x, errX := u.x.eval()
	if errX != nil {
		return makeNone(), errX
	}
	switch u.opSym {
	case "!":
		if x.typ != Boolean {
			return makeNone(), fmt.Errorf(
				"logical negation is only valid for boolean, got %v", x.typ,
			)
		}
		return value{typ: Boolean, govalue: !x.govalue.(bool)}, nil
	case "-":
		if x.typ == Int {
			return value{typ: Int, govalue: -x.govalue.(int64)}, nil
		}
		if x.typ == Float {
			return value{typ: Float, govalue: -x.govalue.(float64)}, nil
		}
		return makeNone(), fmt.Errorf(
			"arithmetic negation is only valid for int or float, got %v",
			x.typ,
		)
	case "+", "()": // positive or expression group
		return x, nil
	default:
		return makeNone(), fmt.Errorf("unknown unary operator: %v", u.opSym)
	}
}

func (u unaryExpr) getOperand() evaluator {
	var evals []evaluator
	for _, child := range u.getChildren() {
		if ce, isExpr := child.(evaluator); isExpr {
			evals = append(evals, ce)
		}
	}
	if len(evals) != 1 {
		log.Fatalf(
			"unary expression expects exactly 1 operand, found %v", len(evals),
		)
	} else {
		return evals[0]
	}
	return nil
}

// A binaryExpr node represents a binary operation.
type binaryExpr struct {
	genNode
	x     evaluator // left operand
	y     evaluator // right operand
	opSym string
}

func newBinaryExpr(start, end int, symbol string) *binaryExpr {
	b := new(binaryExpr)
	b.genNode = genNode{start: start, end: end}
	b.opSym = symbol
	return b
}

func (*binaryExpr) getKind() nodeKind { return exp }

func wdlOr(x, y value) (value, error) {
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

func (b binaryExpr) eval() (value, error) {
	x, errX := b.x.eval()
	if errX != nil {
		return makeNone(), errX
	}
	y, errY := b.y.eval()
	if errY != nil {
		return makeNone(), errY
	}
	switch b.opSym {
	case "||":
		return wdlOr(x, y)
	case "&&":
		return wdlAnd(x, y)
	case "<":
		return wdlLt(x, y)
	case "<=":
		return wdlLe(x, y)
	case ">=":
		return wdlGe(x, y)
	case ">":
		return wdlGt(x, y)
	case "*":
		return wdlMul(x, y)
	case "/":
		return wdlDiv(x, y)
	case "%":
		return wdlMod(x, y)
	case "-":
		return wdlSub(x, y)
	default:
		return makeNone(), fmt.Errorf("unknown binary operator: %v", b.opSym)
	}
}

func (b binaryExpr) getOperands() (evaluator, evaluator) {
	var evals []evaluator
	for _, child := range b.getChildren() {
		if ce, isExpr := child.(evaluator); isExpr {
			evals = append(evals, ce)
		}
	}
	if len(evals) != 2 {
		log.Fatalf(
			"binary expression expects exactly 2 operands, found %v",
			len(evals),
		)
	} else {
		return evals[0], evals[1]
	}
	return nil, nil
}

// Antlr4 listeners

func (l *wdlv1_1Listener) ExitPrimitive_literal(
	ctx *parser.Primitive_literalContext,
) {
	var p *primitiveLiteral
	// BoolLiteral of primitive_literal
	boolToken := ctx.BoolLiteral()
	if boolToken != nil {
		p = newPrimitiveLiteral(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			boolToken.GetText(),
			Boolean,
		)
		l.branching(p, false)
	}
}

func (l *wdlv1_1Listener) ExitNumber(ctx *parser.NumberContext) {
	var p *primitiveLiteral

	// IntLiteral
	intToken := ctx.IntLiteral()
	if intToken != nil {
		p = newPrimitiveLiteral(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			intToken.GetText(),
			Int,
		)
		l.branching(p, false)
	}

	// FloatLiteral
	floatToken := ctx.FloatLiteral()
	if floatToken != nil {
		p = newPrimitiveLiteral(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			floatToken.GetText(),
			Float,
		)
		l.branching(p, false)
	}
}

func (l *wdlv1_1Listener) EnterLor(ctx *parser.LorContext) {
	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.OR().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitLor(ctx *parser.LorContext) {
	lorExpr, isExpr := l.currentNode.(*binaryExpr)
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
	lorExpr.x, lorExpr.y = lorExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterLand(ctx *parser.LandContext) {
	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.AND().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitLand(ctx *parser.LandContext) {
	landExpr, isExpr := l.currentNode.(*binaryExpr)
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
	landExpr.x, landExpr.y = landExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterNegate(ctx *parser.NegateContext) {
	l.branching(
		newUnaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.NOT().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitNegate(ctx *parser.NegateContext) {
	u, isUnaryExpr := l.currentNode.(*unaryExpr)
	if !isUnaryExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"logical NOT",
				"unary expression",
				l.currentNode,
			),
		)
	}
	u.x = u.getOperand()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterUnarysigned(ctx *parser.UnarysignedContext) {
	var opSym string = "+"
	if ctx.MINUS() != nil {
		opSym = "-"
	}
	l.branching(
		newUnaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			opSym,
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitUnarysigned(ctx *parser.UnarysignedContext) {
	u, isUnaryExpr := l.currentNode.(*unaryExpr)
	if !isUnaryExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"positive/negative",
				"unary expression",
				l.currentNode,
			),
		)
	}
	u.x = u.getOperand()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterLt(ctx *parser.LtContext) {
	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.LT().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitLt(ctx *parser.LtContext) {
	ltExpr, isExpr := l.currentNode.(*binaryExpr)
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
	ltExpr.x, ltExpr.y = ltExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterLte(ctx *parser.LteContext) {
	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.LTE().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitLte(ctx *parser.LteContext) {
	lteExpr, isExpr := l.currentNode.(*binaryExpr)
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
	lteExpr.x, lteExpr.y = lteExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterGte(ctx *parser.GteContext) {
	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.GTE().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitGte(ctx *parser.GteContext) {
	gteExpr, isExpr := l.currentNode.(*binaryExpr)
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
	gteExpr.x, gteExpr.y = gteExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterGt(ctx *parser.GtContext) {
	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.GT().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitGt(ctx *parser.GtContext) {
	gtExpr, isExpr := l.currentNode.(*binaryExpr)
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
	gtExpr.x, gtExpr.y = gtExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterMul(ctx *parser.MulContext) {

	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.STAR().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitMul(ctx *parser.MulContext) {
	mulExpr, isExpr := l.currentNode.(*binaryExpr)
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
	mulExpr.x, mulExpr.y = mulExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterDivide(ctx *parser.DivideContext) {
	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.DIVIDE().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitDivide(ctx *parser.DivideContext) {
	divideExpr, isExpr := l.currentNode.(*binaryExpr)
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
	divideExpr.x, divideExpr.y = divideExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterMod(ctx *parser.ModContext) {
	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.MOD().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitMod(ctx *parser.ModContext) {
	modExpr, isExpr := l.currentNode.(*binaryExpr)
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
	modExpr.x, modExpr.y = modExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterSub(ctx *parser.SubContext) {
	l.branching(
		newBinaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			ctx.MINUS().GetText(),
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitSub(ctx *parser.SubContext) {
	subExpr, isExpr := l.currentNode.(*binaryExpr)
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
	subExpr.x, subExpr.y = subExpr.getOperands()
	l.currentNode = l.currentNode.getParent()
}

func (l *wdlv1_1Listener) EnterExpression_group(
	ctx *parser.Expression_groupContext,
) {
	l.branching(
		newUnaryExpr(
			ctx.GetStart().GetStart(),
			ctx.GetStop().GetStop(),
			"()",
		),
		true,
	)
}

func (l *wdlv1_1Listener) ExitExpression_group(
	ctx *parser.Expression_groupContext,
) {
	u, isUnaryExpr := l.currentNode.(*unaryExpr)
	if !isUnaryExpr {
		log.Fatal(
			newMismatchContextError(
				ctx.GetStart().GetLine(),
				ctx.GetStart().GetColumn(),
				"expression group",
				"unary expression",
				l.currentNode,
			),
		)
	}
	u.x = u.getOperand()
	l.currentNode = l.currentNode.getParent()
}
