package wdlparser

import (
	"fmt"
	"log"
	"math"
	"strconv"

	parser "github.com/yunhailuo/wdlparser/pkg/wdlv1_1"
)

type wdlType int

const (
	wdlBoolean wdlType = iota
	wdlInt
	wdlFloat
	wdlAny
)

type wdlValue struct {
	value    interface{} // actually go value
	typ      wdlType
	optional bool
	nonEmpty bool  // array only
	err      error // error message if the actual value is not available or invalid
}

func newWdlBoolean(v interface{}, optional bool, e error) (wdlValue, error) {
	if e == nil && v != nil {
		if _, isBool := v.(bool); !isBool {
			e = fmt.Errorf("input value \"%v\" is not a bool", v)
		}
	}
	return wdlValue{v, wdlBoolean, false, false, e}, e
}

func newWdlInt(v interface{}, optional bool, e error) (wdlValue, error) {
	if e == nil && v != nil {
		switch i := v.(type) {
		case int:
			v = int64(i)
		case int8:
			v = int64(i)
		case int16:
			v = int64(i)
		case int32:
			v = int64(i)
		case int64:
			v = i
		case uint:
			v = int64(i)
		case uint8:
			v = int64(i)
		case uint16:
			v = int64(i)
		case uint32:
			v = int64(i)
		default:
			e = fmt.Errorf(
				"input value \"%v\" is not a go integer"+
					" or potentially too big for int64", v,
			)
		}
	}
	return wdlValue{v, wdlInt, false, false, e}, e
}

func newWdlFloat(v interface{}, optional bool, e error) (wdlValue, error) {
	if e == nil && v != nil {
		switch i := v.(type) {
		case int:
			v = float64(i)
		case int8:
			v = float64(i)
		case int16:
			v = float64(i)
		case int32:
			v = float64(i)
		case int64:
			v = float64(i)
		case uint:
			v = float64(i)
		case uint8:
			v = float64(i)
		case uint16:
			v = float64(i)
		case uint32:
			v = float64(i)
		case uint64:
			v = float64(i)
		case float32:
			v = float64(i)
		case float64:
			v = i
		default:
			e = fmt.Errorf("input value \"%v\" is not a go integer or float", v)
		}
	}
	return wdlValue{v, wdlFloat, false, false, e}, e
}

func wdlNone() wdlValue {
	return wdlValue{nil, wdlAny, false, false, nil}
}

type evaluator func() (wdlValue, error)

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
	e.eval = func() (wdlValue, error) {
		log.Fatalf(
			"\"eval\" function undefined for expression \"%v\" at %d:%d",
			opSym, start, end,
		)
		return wdlNone(), nil
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
		b, err := strconv.ParseBool(boolToken.GetText())
		if err == nil {
			e.eval = func() (wdlValue, error) {
				return newWdlBoolean(b, false, nil)
			}
			l.branching(e, false)
		}
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
		i, err := strconv.ParseInt(intToken.GetText(), 0, 64)
		if err == nil {
			e.eval = func() (wdlValue, error) {
				return newWdlInt(i, false, nil)
			}
			l.branching(e, false)
		}
	}
	// FloatLiteral
	floatToken := ctx.FloatLiteral()
	if floatToken != nil {
		f, err := strconv.ParseFloat(floatToken.GetText(), 64)
		if err == nil {
			e.eval = func() (wdlValue, error) {
				return newWdlFloat(f, false, nil)
			}
			l.branching(e, false)
		}
	}
}

func wdlOr(operands ...wdlValue) (wdlValue, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return wdlNone(), fmt.Errorf(
			"found %d operands, expect 2 for OR operation", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	xIsBool, yIsBool := x.typ == wdlBoolean, y.typ == wdlBoolean
	switch {
	case xIsBool && yIsBool:
		return newWdlBoolean(x.value.(bool) || y.value.(bool), false, nil)
	case xIsBool && (!yIsBool): // only y is invalid
		return wdlNone(), fmt.Errorf(
			"found right operand is not bool, " +
				"expect both operands being bool for OR operation",
		)
	case (!xIsBool) && yIsBool: // only x is invalid
		return wdlNone(), fmt.Errorf(
			"found left operand is not bool, " +
				"expect both operands being bool for OR operation",
		)
	default: // neither operands is valid
		return wdlNone(), fmt.Errorf(
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
	e.eval = func() (wdlValue, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return wdlNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return wdlNone(), errY
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

func wdlAnd(operands ...wdlValue) (wdlValue, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return wdlNone(), fmt.Errorf(
			"found %d operands, expect 2 for AND operation", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	xIsBool, yIsBool := x.typ == wdlBoolean, y.typ == wdlBoolean
	switch {
	case xIsBool && yIsBool:
		return newWdlBoolean(x.value.(bool) && y.value.(bool), false, nil)
	case xIsBool && (!yIsBool): // only y is invalid
		return wdlNone(), fmt.Errorf(
			"found right operand is not bool, " +
				"expect both operands being bool for AND operation",
		)
	case (!xIsBool) && yIsBool: // only x is invalid
		return wdlNone(), fmt.Errorf(
			"found left operand is not bool, " +
				"expect both operands being bool for AND operation",
		)
	default: // neither operands is valid
		return wdlNone(), fmt.Errorf(
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
	e.eval = func() (wdlValue, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return wdlNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return wdlNone(), errY
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

func wdlNegate(operands ...wdlValue) (wdlValue, error) {
	operandCount := len(operands)
	if operandCount != 1 {
		return wdlNone(), fmt.Errorf(
			"found %d operands, expect 1 for negation", operandCount,
		)
	}
	x := operands[0]
	switch x.typ {
	case wdlBoolean:
		return newWdlBoolean(!x.value.(bool), false, nil)
	case wdlInt:
		return newWdlInt(-x.value.(int64), false, nil)
	case wdlFloat:
		return newWdlFloat(-x.value.(float64), false, nil)
	default:
		return wdlNone(), fmt.Errorf(
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
	e.eval = func() (wdlValue, error) {
		y, errY := e.y.eval()
		if errY != nil {
			return wdlNone(), errY
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
	e.eval = func() (wdlValue, error) {
		y, errY := e.y.eval()
		if errY != nil {
			return wdlNone(), errY
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

func wdlMul(operands ...wdlValue) (wdlValue, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return wdlNone(), fmt.Errorf(
			"found %d operands, expect 2 for multiplication", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == wdlInt && y.typ == wdlInt:
		return newWdlInt((x.value.(int64))*(y.value.(int64)), false, nil)
	case x.typ == wdlFloat && y.typ == wdlInt:
		return newWdlFloat(
			(x.value.(float64))*float64(y.value.(int64)), false, nil,
		)
	case x.typ == wdlInt && y.typ == wdlFloat:
		return newWdlFloat(
			float64(x.value.(int64))*(y.value.(float64)), false, nil,
		)
	case x.typ == wdlFloat && y.typ == wdlFloat:
		return newWdlFloat(
			(x.value.(float64))*(y.value.(float64)), false, nil,
		)
	default: // neither operands is int or float
		return wdlNone(), fmt.Errorf(
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
	e.eval = func() (wdlValue, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return wdlNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return wdlNone(), errY
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

func wdlDiv(operands ...wdlValue) (wdlValue, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return wdlNone(), fmt.Errorf(
			"found %d operands, expect 2 for division", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == wdlInt && y.typ == wdlInt:
		return newWdlInt((x.value.(int64))/(y.value.(int64)), false, nil)
	case x.typ == wdlFloat && y.typ == wdlInt:
		return newWdlFloat(
			(x.value.(float64))/float64(y.value.(int64)), false, nil,
		)
	case x.typ == wdlInt && y.typ == wdlFloat:
		return newWdlFloat(
			float64(x.value.(int64))/(y.value.(float64)), false, nil,
		)
	case x.typ == wdlFloat && y.typ == wdlFloat:
		return newWdlFloat(
			(x.value.(float64))/(y.value.(float64)), false, nil,
		)
	default: // neither operands is int or float
		return wdlNone(), fmt.Errorf(
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
	e.eval = func() (wdlValue, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return wdlNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return wdlNone(), errY
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

func wdlMod(operands ...wdlValue) (wdlValue, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return wdlNone(), fmt.Errorf(
			"found %d operands, expect 2 for modulo", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == wdlInt && y.typ == wdlInt:
		return newWdlInt((x.value.(int64))%(y.value.(int64)), false, nil)
	case x.typ == wdlFloat && y.typ == wdlInt:
		return newWdlFloat(
			math.Mod(x.value.(float64), float64(y.value.(int64))), false, nil,
		)
	case x.typ == wdlInt && y.typ == wdlFloat:
		return newWdlFloat(
			math.Mod(float64(x.value.(int64)), y.value.(float64)), false, nil,
		)
	case x.typ == wdlFloat && y.typ == wdlFloat:
		return newWdlFloat(
			math.Mod(x.value.(float64), y.value.(float64)), false, nil,
		)
	default: // neither operands is int or float
		return wdlNone(), fmt.Errorf(
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
	e.eval = func() (wdlValue, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return wdlNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return wdlNone(), errY
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

func wdlSub(operands ...wdlValue) (wdlValue, error) {
	operandCount := len(operands)
	if operandCount != 2 {
		return wdlNone(), fmt.Errorf(
			"found %d operands, expect 2 for subtraction", operandCount,
		)
	}
	x, y := operands[0], operands[1]
	switch {
	case x.typ == wdlInt && y.typ == wdlInt:
		return newWdlInt((x.value.(int64))-(y.value.(int64)), false, nil)
	case x.typ == wdlFloat && y.typ == wdlInt:
		return newWdlFloat(
			(x.value.(float64))-float64(y.value.(int64)), false, nil,
		)
	case x.typ == wdlInt && y.typ == wdlFloat:
		return newWdlFloat(
			float64(x.value.(int64))-(y.value.(float64)), false, nil,
		)
	case x.typ == wdlFloat && y.typ == wdlFloat:
		return newWdlFloat(
			(x.value.(float64))-(y.value.(float64)), false, nil,
		)
	default: // neither operands is int or float
		return wdlNone(), fmt.Errorf(
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
	e.eval = func() (wdlValue, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return wdlNone(), errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return wdlNone(), errY
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
	e.eval = func() (wdlValue, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return wdlNone(), errX
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
