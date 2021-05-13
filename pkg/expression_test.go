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
