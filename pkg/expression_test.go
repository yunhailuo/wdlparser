package wdlparser

import (
	"testing"
)

func TestBoolLiteralOrAndNot(t *testing.T) {
	testCases := []struct {
		wdl  string
		want bool
	}{
		{"version 1.1 workflow L {input{Boolean t=true || true}}", true},
		{"version 1.1 workflow L {input{Boolean t=true || false}}", true},
		{"version 1.1 workflow L {input{Boolean t=false || true}}", true},
		{"version 1.1 workflow L {input{Boolean t=false || false}}", false},
		{"version 1.1 workflow L {input{Boolean t=true && true}}", true},
		{"version 1.1 workflow L {input{Boolean t=true && false}}", false},
		{"version 1.1 workflow L {input{Boolean t=false && true}}", false},
		{"version 1.1 workflow L {input{Boolean t=false && false}}", false},
		{"version 1.1 workflow L {input{Boolean t=!true}}", false},
		{"version 1.1 workflow L {input{Boolean t=!false}}", true},
	}
	for _, tc := range testCases {
		result, err := Antlr4Parse(tc.wdl)
		if err != nil {
			t.Errorf(
				"Found %d errors in %q, expect no errors", len(err), tc.wdl,
			)
		}
		v, evalErr := result.Workflow.Inputs[0].expr.eval()
		if evalErr != nil {
			t.Errorf("Fail to evaluate %v: %w", tc.wdl, evalErr)
		}
		if v.govalue != tc.want {
			t.Errorf("Evaluate %v as %v, expect %t", tc.wdl, v.govalue, tc.want)
		}
	}
}

func TestComparison(t *testing.T) {
	testCases := []struct {
		wdl  string
		want bool
	}{
		// Less than, less than or equal to
		{"version 1.1 workflow L {input{Boolean t=1<2}}", true},
		{"version 1.1 workflow L {input{Boolean t=1.0<=2.0}}", true},
		{"version 1.1 workflow L {input{Boolean t=3.0<3}}", false},
		{"version 1.1 workflow L {input{Boolean t=3<=3.0}}", true},
		{"version 1.1 workflow L {input{Boolean t=5.0<4}}", false},
		{"version 1.1 workflow L {input{Boolean t=5<=4}}", false},
		{"version 1.1 workflow L {input{Boolean t=6.0<10.0}}", true},
		{"version 1.1 workflow L {input{Boolean t=6.0<=10}}", true},

		// Greater than, greater than or equal to
		{"version 1.1 workflow L {input{Boolean t=1>2}}", false},
		{"version 1.1 workflow L {input{Boolean t=1.0>=2.0}}", false},
		{"version 1.1 workflow L {input{Boolean t=3.0>3}}", false},
		{"version 1.1 workflow L {input{Boolean t=3>=3.0}}", true},
		{"version 1.1 workflow L {input{Boolean t=5.0>4}}", true},
		{"version 1.1 workflow L {input{Boolean t=5>=4}}", true},
		{"version 1.1 workflow L {input{Boolean t=6.0>10.0}}", false},
		{"version 1.1 workflow L {input{Boolean t=6.0>=10}}", false},
	}
	for _, tc := range testCases {
		result, err := Antlr4Parse(tc.wdl)
		if err != nil {
			t.Errorf(
				"Found %d errors in %q, expect no errors", len(err), tc.wdl,
			)
		}
		v, evalErr := result.Workflow.Inputs[0].expr.eval()
		if evalErr != nil {
			t.Errorf("Fail to evaluate %v: %w", tc.wdl, evalErr)
		}
		if v.govalue != tc.want {
			t.Errorf("Evaluate %v as %v, expect %v", tc.wdl, v.govalue, tc.want)
		}
	}
}

func TestArithmetic(t *testing.T) {
	testCases := []struct {
		wdl  string
		want interface{}
	}{
		{"version 1.1 workflow L {input{Int t=-2}}", int64(-2)},
		{"version 1.1 workflow L {input{Int t=+2}}", int64(2)},
		{"version 1.1 workflow L {input{Float t=-2.0}}", -2.0},
		{"version 1.1 workflow L {input{Float t=+2.0}}", 2.0},

		// Substraction
		{"version 1.1 workflow L {input{Int t=3-1}}", int64(2)},
		{"version 1.1 workflow L {input{Float t=5.0-4.0}}", 1.0},
		{"version 1.1 workflow L {input{Float t=10-6.0}}", 4.0},
		{"version 1.1 workflow L {input{Float t=0.0-2}}", -2.0},

		// Multiplication
		{"version 1.1 workflow L {input{Int t=2*3}}", int64(6)},
		{"version 1.1 workflow L {input{Float t=5.0*4.0}}", 20.0},
		{"version 1.1 workflow L {input{Float t=7*6.0}}", 42.0},
		{"version 1.1 workflow L {input{Float t=8.0*9}}", 72.0},

		// Division
		{"version 1.1 workflow L {input{Int t=4/2}}", int64(2)},
		{"version 1.1 workflow L {input{Float t=6.0/3.0}}", 2.0},
		{"version 1.1 workflow L {input{Float t=8/2.0}}", 4.0},
		{"version 1.1 workflow L {input{Float t=2.0/10}}", 0.2},

		// Modulo
		{"version 1.1 workflow L {input{Int t=3%2}}", int64(1)},
		{"version 1.1 workflow L {input{Float t=5.2%3.0}}", 2.2},
		{"version 1.1 workflow L {input{Float t=10%6.3}}", 3.7},
		{"version 1.1 workflow L {input{Float t=0.0%2}}", 0.0},
	}
	for _, tc := range testCases {
		result, err := Antlr4Parse(tc.wdl)
		if err != nil {
			t.Errorf(
				"Found %d errors in %q, expect no errors", len(err), tc.wdl,
			)
		}
		v, evalErr := result.Workflow.Inputs[0].expr.eval()
		if evalErr != nil {
			t.Errorf("Fail to evaluate %v: %w", tc.wdl, evalErr)
		}
		if v.govalue != tc.want {
			t.Errorf("Evaluate %v as %v, expect %v", tc.wdl, v.govalue, tc.want)
		}
	}
}
