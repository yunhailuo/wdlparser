package wdlparser

import (
	"fmt"
	"log"
	"math"
	"strconv"

	parser "github.com/yunhailuo/wdlparser/pkg/wdlv1_1"
)

type evaluator func() (interface{}, error)

func newLiteralEval(literal interface{}) evaluator {
	return func() (interface{}, error) {
		return literal, nil
	}
}

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
	e.eval = func() (interface{}, error) {
		log.Fatalf(
			"\"eval\" function undefined for expression \"%v\" at %d:%d",
			opSym, start, end,
		)
		return nil, nil
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
			e.eval = newLiteralEval(b)
			l.branching(e, false)
		}
	}
}

func (l *wdlv1_1Listener) EnterLor(ctx *parser.LorContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.OR().GetText(),
	)
	e.eval = func() (interface{}, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return nil, errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return nil, errY
		}
		xVal, xIsBool := x.(bool)
		yVal, yIsBool := y.(bool)
		if (!xIsBool) || (!yIsBool) {
			return nil, fmt.Errorf(
				"invalid operands for OR at %d:%d: found \"%T || %T\","+
					" expect \"bool || bool\"",
				e.getStart(), e.getEnd(), x, y,
			)
		}
		return xVal || yVal, nil
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

func (l *wdlv1_1Listener) EnterLand(ctx *parser.LandContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.AND().GetText(),
	)
	e.eval = func() (interface{}, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return nil, errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return nil, errY
		}
		xVal, xIsBool := x.(bool)
		yVal, yIsBool := y.(bool)
		if (!xIsBool) || (!yIsBool) {
			return nil, fmt.Errorf(
				"invalid operands for AND at %d:%d: found \"%T && %T\","+
					" expect \"bool && bool\"",
				e.getStart(), e.getEnd(), x, y,
			)
		}
		return xVal && yVal, nil
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

func (l *wdlv1_1Listener) EnterNegate(ctx *parser.NegateContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.NOT().GetText(),
	)
	e.eval = func() (interface{}, error) {
		y, errY := e.y.eval()
		if errY != nil {
			return nil, errY
		}
		yVal, yIsBool := y.(bool)
		if !yIsBool {
			return nil, fmt.Errorf(
				"invalid operands for NOT at %d:%d: found \"!%T\","+
					" expect \"!bool\"",
				e.getStart(), e.getEnd(), y,
			)
		}
		return !yVal, nil
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
			e.eval = newLiteralEval(i)
			l.branching(e, false)
		}
	}
	// FloatLiteral
	floatToken := ctx.FloatLiteral()
	if floatToken != nil {
		f, err := strconv.ParseFloat(floatToken.GetText(), 64)
		if err == nil {
			e.eval = newLiteralEval(f)
			l.branching(e, false)
		}
	}
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
	e.eval = func() (interface{}, error) {
		y, errY := e.y.eval()
		if errY != nil {
			return nil, errY
		}
		yInt, yIsInt64 := y.(int64)
		yFloat, yIsFloat64 := y.(float64)
		if (!yIsInt64) && (!yIsFloat64) {
			return nil, fmt.Errorf(
				"invalid operands for %v at %d:%d: found \"%v %T\","+
					" expect \"%v int/float\"",
				opSym, e.getStart(), e.getEnd(), opSym, y, opSym,
			)
		}
		switch {
		case yIsInt64:
			if opSym == "+" {
				return yInt, nil
			}
			// opSym == "-"
			return -yInt, nil
		default: // yIsFloat64:
			if opSym == "+" {
				return yFloat, nil
			}
			// opSym == "-"
			return -yFloat, nil
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

func (l *wdlv1_1Listener) EnterMul(ctx *parser.MulContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.STAR().GetText(),
	)
	e.eval = func() (interface{}, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return nil, errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return nil, errY
		}
		xInt, xIsInt64 := x.(int64)
		xFloat, xIsFloat64 := x.(float64)
		yInt, yIsInt64 := y.(int64)
		yFloat, yIsFloat64 := y.(float64)
		if ((!xIsInt64) && (!xIsFloat64)) || ((!yIsInt64) && (!yIsFloat64)) {
			return nil, fmt.Errorf(
				"invalid operands for multiply at %d:%d: found \"%T * %T\","+
					" expect \"int/float * int/float\"",
				e.getStart(), e.getEnd(), x, y,
			)
		}
		switch {
		case xIsInt64 && yIsInt64:
			return xInt * yInt, nil
		case xIsInt64 && yIsFloat64:
			return float64(xInt) * yFloat, nil
		case xIsFloat64 && yIsInt64:
			return xFloat * float64(yInt), nil
		default: // xIsFloat64 && yIsFloat64:
			return xFloat * yFloat, nil
		}
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

func (l *wdlv1_1Listener) EnterDivide(ctx *parser.DivideContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.DIVIDE().GetText(),
	)
	e.eval = func() (interface{}, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return nil, errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return nil, errY
		}
		xInt, xIsInt64 := x.(int64)
		xFloat, xIsFloat64 := x.(float64)
		yInt, yIsInt64 := y.(int64)
		yFloat, yIsFloat64 := y.(float64)
		if ((!xIsInt64) && (!xIsFloat64)) || ((!yIsInt64) && (!yIsFloat64)) {
			return nil, fmt.Errorf(
				"invalid operands for divide at %d:%d: found \"%T / %T\","+
					" expect \"int/float / int/float\"",
				e.getStart(), e.getEnd(), x, y,
			)
		}
		switch {
		case xIsInt64 && yIsInt64:
			return xInt / yInt, nil
		case xIsInt64 && yIsFloat64:
			return float64(xInt) / yFloat, nil
		case xIsFloat64 && yIsInt64:
			return xFloat / float64(yInt), nil
		default: // xIsFloat64 && yIsFloat64:
			return xFloat / yFloat, nil
		}
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

func (l *wdlv1_1Listener) EnterMod(ctx *parser.ModContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.MOD().GetText(),
	)
	e.eval = func() (interface{}, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return nil, errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return nil, errY
		}
		xInt, xIsInt64 := x.(int64)
		xFloat, xIsFloat64 := x.(float64)
		yInt, yIsInt64 := y.(int64)
		yFloat, yIsFloat64 := y.(float64)
		if ((!xIsInt64) && (!xIsFloat64)) || ((!yIsInt64) && (!yIsFloat64)) {
			return nil, fmt.Errorf(
				"invalid operands for modulo at %d:%d: found \"%T / %T\","+
					" expect \"int/float / int/float\"",
				e.getStart(), e.getEnd(), x, y,
			)
		}
		switch {
		case xIsInt64 && yIsInt64:
			return xInt % yInt, nil
		case xIsInt64 && yIsFloat64:
			return math.Mod(float64(xInt), yFloat), nil
		case xIsFloat64 && yIsInt64:
			return math.Mod(xFloat, float64(yInt)), nil
		default: // xIsFloat64 && yIsFloat64:
			return math.Mod(xFloat, yFloat), nil
		}
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

func (l *wdlv1_1Listener) EnterSub(ctx *parser.SubContext) {
	e := newExpr(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		ctx.MINUS().GetText(),
	)
	e.eval = func() (interface{}, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return nil, errX
		}
		y, errY := e.y.eval()
		if errY != nil {
			return nil, errY
		}
		xInt, xIsInt64 := x.(int64)
		xFloat, xIsFloat64 := x.(float64)
		yInt, yIsInt64 := y.(int64)
		yFloat, yIsFloat64 := y.(float64)
		if ((!xIsInt64) && (!xIsFloat64)) || ((!yIsInt64) && (!yIsFloat64)) {
			return nil, fmt.Errorf(
				"invalid operands for subtract at %d:%d: found \"%T - %T\","+
					" expect \"int/float - int/float\"",
				e.getStart(), e.getEnd(), x, y,
			)
		}
		switch {
		case xIsInt64 && yIsInt64:
			return xInt - yInt, nil
		case xIsInt64 && yIsFloat64:
			return float64(xInt) - yFloat, nil
		case xIsFloat64 && yIsInt64:
			return xFloat - float64(yInt), nil
		default: // xIsFloat64 && yIsFloat64:
			return xFloat - yFloat, nil
		}
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
	e.eval = func() (interface{}, error) {
		x, errX := e.x.eval()
		if errX != nil {
			return nil, errX
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
