package wdlparser

import (
	"reflect"
	"testing"
)

func TestSinglePrimitiveExpression(t *testing.T) {
	testCases := []struct {
		wdl  string
		want interface{}
	}{
		{
			"version 1.1 workflow Test {input{Int t=-3}}",
			[]interface{}{value{Int, int64(3)}, wdlNeg},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=!true}}",
			[]interface{}{value{Boolean, true}, wdlNot},
		},
		{
			"version 1.1 workflow Test {input{Int t=3.0/4.0}}",
			[]interface{}{
				value{Float, float64(3)}, value{Float, float64(4)}, wdlDiv,
			},
		},
		{
			"version 1.1 workflow Test {input{Int t=3%4}}",
			[]interface{}{value{Int, int64(3)}, value{Int, int64(4)}, wdlMod},
		},
		{
			"version 1.1 workflow Test {input{Int t=3+4}}",
			[]interface{}{value{Int, int64(3)}, value{Int, int64(4)}, wdlAdd},
		},
		{
			"version 1.1 workflow Test {input{Int t=3.0-4}}",
			[]interface{}{
				value{Float, float64(3)}, value{Int, int64(4)}, wdlSub,
			},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true==false}}",
			[]interface{}{value{Boolean, true}, value{Boolean, false}, wdlEq},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true!=false}}",
			[]interface{}{value{Boolean, true}, value{Boolean, false}, wdlNeq},
		},
		{
			"version 1.1 workflow Test {input{Int t=3<4}}",
			[]interface{}{value{Int, int64(3)}, value{Int, int64(4)}, wdlLt},
		},
		{
			"version 1.1 workflow Test {input{Int t=3<=4}}",
			[]interface{}{value{Int, int64(3)}, value{Int, int64(4)}, wdlLte},
		},
		{
			"version 1.1 workflow Test {input{Int t=3>4}}",
			[]interface{}{value{Int, int64(3)}, value{Int, int64(4)}, wdlGt},
		},
		{
			"version 1.1 workflow Test {input{Int t=3>=4}}",
			[]interface{}{value{Int, int64(3)}, value{Int, int64(4)}, wdlGte},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true&&false}}",
			[]interface{}{value{Boolean, true}, value{Boolean, false}, wdlAnd},
		},
		{
			"version 1.1 workflow Test {input{Boolean t=true||false}}",
			[]interface{}{value{Boolean, true}, value{Boolean, false}, wdlOr},
		},
	}
	for _, tc := range testCases {
		result, err := Antlr4Parse(tc.wdl)
		if err != nil {
			t.Errorf(
				"Found %d errors in %q, expect no errors", len(err), tc.wdl,
			)
		}
		v := result.Workflow.Inputs.decls[0].initialization
		if !reflect.DeepEqual(v.rpn, tc.want) {
			t.Errorf("Evaluate %v as %v, expect %v", tc.wdl, v.rpn, tc.want)
		}
	}
}
