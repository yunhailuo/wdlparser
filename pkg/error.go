package wdlparser

import (
	"fmt"

	"github.com/antlr/antlr4/runtime/Go/antlr"
)

// WDLSyntaxError is used to store WDL error line, column and details of a
// syntax error
type WDLSyntaxError struct {
	line, column int
	msg          string
}

func (e WDLSyntaxError) Error() string {
	return fmt.Sprintf("line %d:%d %q", e.line, e.column, e.msg)
}

func NewWDLSyntaxError(line, column int, msg string) WDLSyntaxError {
	return WDLSyntaxError{line, column, msg}
}

type wdlErrorListener struct {
	*antlr.DiagnosticErrorListener
	syntaxErrors []WDLSyntaxError
}

func newWdlErrorListener(exactOnly bool) *wdlErrorListener {
	return &wdlErrorListener{antlr.NewDiagnosticErrorListener(exactOnly), nil}
}

func (l *wdlErrorListener) SyntaxError(
	recognizer antlr.Recognizer,
	offendingSymbol interface{},
	line, column int,
	msg string,
	e antlr.RecognitionException,
) {
	l.syntaxErrors = append(
		l.syntaxErrors, NewWDLSyntaxError(line, column, msg),
	)
}
