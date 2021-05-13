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
	start, end int
	kind       nodeKind
	parent     node
	children   []node

	x *expr // left operand
	y *expr // right operand

	opSym string
	eval  evaluator // evaluate this expression (such as adding [x] to [y])
}

func newExpr(start, end int, opSym string) *expr {
	e := new(expr)
	e.start = start
	e.end = end
	e.kind = exp
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

func (e *expr) getStart() int {
	return e.start
}

func (e *expr) getEnd() int {
	return e.end
}

func (e *expr) getKind() nodeKind {
	return exp
}

func (e *expr) setKind(kind nodeKind) {}

func (e *expr) getParent() node {
	return e.parent
}

func (e *expr) setParent(parent node) {
	e.parent = parent
	parent.addChild(e)
}

func (e *expr) getChildren() []node {
	return e.children
}

func (e *expr) addChild(n node) {
	newStart := n.getStart()
	newEnd := n.getEnd()
	for _, child := range e.children {
		if (child.getStart() == newStart) && (child.getEnd() == newEnd) {
			return
		}
	}
	e.children = append(e.children, n)
	// Note that this add child method will not set parent on node `n`
}

func (e *expr) getChildExprs() []*expr {
	var exprs []*expr
	for _, child := range e.getChildren() {
		if ce, ok := child.(*expr); ok {
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
		xVal, xOk := x.(bool)
		if !xOk {
			return nil, fmt.Errorf(
				"left operand, %v, of OR at %d:%d is not a valid bool",
				x, e.getStart(), e.getEnd(),
			)
		}
		yVal, yOk := y.(bool)
		if !yOk {
			return nil, fmt.Errorf(
				"right operand, %v, of OR at %d:%d is not a valid bool",
				y, e.getStart(), e.getEnd(),
			)
		}
		return xVal || yVal, nil
	}
	e.setParent(l.currentNode)
	l.currentNode = e
}

func (l *wdlv1_1Listener) ExitLor(ctx *parser.LorContext) {
	lorExpr, ok := l.currentNode.(*expr)
	if !ok {
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
		xVal, xOk := x.(bool)
		if !xOk {
			return nil, fmt.Errorf(
				"left operand, %v, of AND at %d:%d is not a valid bool",
				x, e.getStart(), e.getEnd(),
			)
		}
		yVal, yOk := y.(bool)
		if !yOk {
			return nil, fmt.Errorf(
				"right operand, %v, of AND at %d:%d is not a valid bool",
				y, e.getStart(), e.getEnd(),
			)
		}
		return xVal && yVal, nil
	}
	e.setParent(l.currentNode)
	l.currentNode = e
}

func (l *wdlv1_1Listener) ExitLand(ctx *parser.LandContext) {
	lorExpr, ok := l.currentNode.(*expr)
	if !ok {
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
			e.setParent(l.currentNode)
		}
	}
}
