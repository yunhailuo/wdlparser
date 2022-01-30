package wdlparser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPrimitiveLiteral(t *testing.T) {
	testCases := []struct {
		wdl  string
		want interface{}
	}{
		{
			"version 1.1 workflow Test {input{Int t=identifier}}",
			exprRPN{newIdentifier("identifier", true)},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true}}",
			exprRPN{value{Boolean, true}},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=false}}",
			exprRPN{value{Boolean, false}},
		},
		{
			"version 1.1 workflow Test {input{Int t=00}}",
			exprRPN{value{Int, int64(0)}},
		},
		{
			"version 1.1 workflow Test {input{Float t=0.0}}",
			exprRPN{value{Float, float64(0.0)}},
		},
		{
			"version 1.1 workflow Test {input{String t='single quote string'}}",
			exprRPN{value{String, "single quote string"}},
		},
		{
			`version 1.1 workflow Test {input{String t="double quote string"}}`,
			exprRPN{value{String, "double quote string"}},
		},
	}
	for _, tc := range testCases {
		result, err := Antlr4Parse(tc.wdl)
		if err != nil {
			t.Errorf(
				"Found %d errors in %q, expect no errors", len(err), tc.wdl,
			)
		}
		v := *result.Workflow.Inputs[0].value
		if diff := cmp.Diff(tc.want, v, commonCmpopts...); diff != "" {
			t.Errorf("unexpected workflow calls:\n%s", diff)
		}
	}
}

func TestExpressionPlaceholder(t *testing.T) {
	testCases := []struct {
		wdl  string
		want interface{}
	}{
		{
			`version 1.1 workflow Test {input{String t="~{1 + i}"}}`,
			exprRPN{
				value{String, ""},
				&expression{
					genNode: genNode{start: 45, end: 49},
					rpn: exprRPN{
						value{Int, int64(1)},
						newIdentifier("i", true),
						WDLAdd,
					},
				},
				WDLStr,
				value{String, ""},
				WDLAdd,
				WDLAdd,
			},
		},
		{
			`version 1.1 workflow Test ` +
				`{input{String t="grep '~{start}...~{end}' ~{file}"}}`,
			exprRPN{
				value{String, "grep '"},
				&expression{
					genNode: genNode{start: 51, end: 55},
					rpn:     exprRPN{newIdentifier("start", true)},
				},
				WDLStr,
				value{String, "..."},
				WDLAdd,
				WDLAdd,
				&expression{
					genNode: genNode{start: 62, end: 64},
					rpn:     exprRPN{newIdentifier("end", true)},
				},
				WDLStr,
				value{String, "' "},
				WDLAdd,
				WDLAdd,
				&expression{
					genNode: genNode{start: 70, end: 73},
					rpn:     exprRPN{newIdentifier("file", true)},
				},
				WDLStr,
				value{String, ""},
				WDLAdd,
				WDLAdd,
			},
		},
	}
	for _, tc := range testCases {
		result, err := Antlr4Parse(tc.wdl)
		if err != nil {
			t.Errorf(
				"Found %d errors in %q, expect no errors", len(err), tc.wdl,
			)
		}
		v := *result.Workflow.Inputs[0].value
		if diff := cmp.Diff(tc.want, v, commonCmpopts...); diff != "" {
			t.Errorf("unexpected workflow calls:\n%s", diff)
		}
	}
}

func TestSinglePrimitiveExpression(t *testing.T) {
	testCases := []struct {
		wdl  string
		want interface{}
	}{
		{
			"version 1.1 workflow Test {input{Int t=-3}}",
			exprRPN{
				&expression{
					genNode: genNode{start: 40, end: 40},
					rpn:     exprRPN{value{Int, int64(3)}},
				},
				WDLNeg,
			},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=!true}}",
			exprRPN{
				&expression{
					genNode: genNode{start: 44, end: 47},
					rpn:     exprRPN{value{Boolean, true}},
				},
				WDLNot,
			},
		},
		{
			"version 1.1 workflow Test {input{Int t=3.0/4.0}}",
			exprRPN{
				value{Float, float64(3)}, value{Float, float64(4)}, WDLDiv,
			},
		},
		{
			"version 1.1 workflow Test {input{Int t=3%4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, WDLMod},
		},
		{
			"version 1.1 workflow Test {input{Int t=3+4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, WDLAdd},
		},
		{
			"version 1.1 workflow Test {input{Int t=3.0-4}}",
			exprRPN{
				value{Float, float64(3)}, value{Int, int64(4)}, WDLSub,
			},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true==false}}",
			exprRPN{value{Boolean, true}, value{Boolean, false}, WDLEq},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true!=false}}",
			exprRPN{value{Boolean, true}, value{Boolean, false}, WDLNeq},
		},
		{
			"version 1.1 workflow Test {input{Int t=3<4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, WDLLt},
		},
		{
			"version 1.1 workflow Test {input{Int t=3<=4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, WDLLte},
		},
		{
			"version 1.1 workflow Test {input{Int t=3>4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, WDLGt},
		},
		{
			"version 1.1 workflow Test {input{Int t=3>=4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, WDLGte},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true&&false}}",
			exprRPN{value{Boolean, true}, value{Boolean, false}, WDLAnd},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true||false}}",
			exprRPN{value{Boolean, true}, value{Boolean, false}, WDLOr},
		},
	}
	for _, tc := range testCases {
		result, err := Antlr4Parse(tc.wdl)
		if err != nil {
			t.Errorf(
				"Found %d errors in %q, expect no errors", len(err), tc.wdl,
			)
		}
		v := *result.Workflow.Inputs[0].value
		if diff := cmp.Diff(tc.want, v, commonCmpopts...); diff != "" {
			t.Errorf("unexpected workflow calls:\n%s", diff)
		}
	}
}

func TestExpression(t *testing.T) {
	testCases := []struct {
		wdl  string
		want interface{}
	}{
		// Substraction
		{
			"version 1.1 workflow Test {input{Int t=3+4*2/(1-5*2)+3}}",
			exprRPN{
				value{Int, int64(3)},
				value{Int, int64(4)},
				value{Int, int64(2)},
				WDLMul,
				&expression{
					genNode: genNode{start: 46, end: 50},
					rpn: exprRPN{value{Int, int64(1)},
						value{Int, int64(5)},
						value{Int, int64(2)},
						WDLMul,
						WDLSub,
					},
				},
				WDLDiv,
				WDLAdd,
				value{Int, int64(3)},
				WDLAdd,
			},
		},
	}
	for _, tc := range testCases {
		result, err := Antlr4Parse(tc.wdl)
		if err != nil {
			t.Errorf(
				"Found %d errors in %q, expect no errors", len(err), tc.wdl,
			)
		}
		v := *result.Workflow.Inputs[0].value
		if diff := cmp.Diff(tc.want, v, commonCmpopts...); diff != "" {
			t.Errorf("unexpected workflow calls:\n%s", diff)
		}
	}
}
