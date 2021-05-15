package wdlparser

import (
	"testing"
)

func TestBoolLiteralOrAnd(t *testing.T) {
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
		if v != tc.want {
			t.Errorf("Evaluate %v as %v, expect %t", tc.wdl, v, tc.want)
		}
	}
}

func TestSubstract(t *testing.T) {
	testCases := []struct {
		wdl  string
		want interface{}
	}{
		{"version 1.1 workflow L {input{Int t=3-1}}", int64(2)},
		{"version 1.1 workflow L {input{Float t=5.0-4.0}}", 1.0},
		{"version 1.1 workflow L {input{Float t=10-6.0}}", 4.0},
		{"version 1.1 workflow L {input{Float t=0.0-2}}", -2.0},
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
		if v != tc.want {
			t.Errorf("Evaluate %v as %T, expect %T", tc.wdl, v, tc.want)
		}
	}
}
