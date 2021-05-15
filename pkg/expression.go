package wdlparser

import (
	"fmt"
	"log"
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
		if !xIsBool {
			return nil, fmt.Errorf(
				"left operand, %v, of OR at %d:%d is not a valid bool",
				x, e.getStart(), e.getEnd(),
			)
		}
		yVal, yIsBool := y.(bool)
		if !yIsBool {
			return nil, fmt.Errorf(
				"right operand, %v, of OR at %d:%d is not a valid bool",
				y, e.getStart(), e.getEnd(),
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
		if !xIsBool {
			return nil, fmt.Errorf(
				"left operand, %v, of AND at %d:%d is not a valid bool",
				x, e.getStart(), e.getEnd(),
			)
		}
		yVal, yIsBool := y.(bool)
		if !yIsBool {
			return nil, fmt.Errorf(
				"right operand, %v, of AND at %d:%d is not a valid bool",
				y, e.getStart(), e.getEnd(),
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
