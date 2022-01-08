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
				&exprRPN{
					value{Int, int64(1)},
					newIdentifier("i", true),
					wdlAdd,
				},
				value{String, ""},
				wdlAdd,
				wdlAdd,
			},
		},
		{
			`version 1.1 workflow Test ` +
				`{input{String t="grep '~{start}...~{end}' ~{file}"}}`,
			exprRPN{
				value{String, "grep '"},
				&exprRPN{newIdentifier("start", true)},
				value{String, "..."},
				wdlAdd,
				wdlAdd,
				&exprRPN{newIdentifier("end", true)},
				value{String, "' "},
				wdlAdd,
				wdlAdd,
				&exprRPN{newIdentifier("file", true)},
				value{String, ""},
				wdlAdd,
				wdlAdd,
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
			exprRPN{&exprRPN{value{Int, int64(3)}}, wdlNeg},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=!true}}",
			exprRPN{&exprRPN{value{Boolean, true}}, wdlNot},
		},
		{
			"version 1.1 workflow Test {input{Int t=3.0/4.0}}",
			exprRPN{
				value{Float, float64(3)}, value{Float, float64(4)}, wdlDiv,
			},
		},
		{
			"version 1.1 workflow Test {input{Int t=3%4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, wdlMod},
		},
		{
			"version 1.1 workflow Test {input{Int t=3+4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, wdlAdd},
		},
		{
			"version 1.1 workflow Test {input{Int t=3.0-4}}",
			exprRPN{
				value{Float, float64(3)}, value{Int, int64(4)}, wdlSub,
			},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true==false}}",
			exprRPN{value{Boolean, true}, value{Boolean, false}, wdlEq},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true!=false}}",
			exprRPN{value{Boolean, true}, value{Boolean, false}, wdlNeq},
		},
		{
			"version 1.1 workflow Test {input{Int t=3<4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, wdlLt},
		},
		{
			"version 1.1 workflow Test {input{Int t=3<=4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, wdlLte},
		},
		{
			"version 1.1 workflow Test {input{Int t=3>4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, wdlGt},
		},
		{
			"version 1.1 workflow Test {input{Int t=3>=4}}",
			exprRPN{value{Int, int64(3)}, value{Int, int64(4)}, wdlGte},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true&&false}}",
			exprRPN{value{Boolean, true}, value{Boolean, false}, wdlAnd},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true||false}}",
			exprRPN{value{Boolean, true}, value{Boolean, false}, wdlOr},
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
				wdlMul,
				&exprRPN{value{Int, int64(1)},
					value{Int, int64(5)},
					value{Int, int64(2)},
					wdlMul,
					wdlSub,
				},
				wdlDiv,
				wdlAdd,
				value{Int, int64(3)},
				wdlAdd,
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
