package wdlparser

import (
	"fmt"

	"github.com/antlr/antlr4/runtime/Go/antlr"
)

// wdlSyntaxError is used to store WDL error line, column and details of a
// syntax error
type wdlSyntaxError struct {
	line, column int
	msg          string
}

func (e wdlSyntaxError) Error() string {
	return fmt.Sprintf("line %d:%d %q", e.line, e.column, e.msg)
}

func newWdlSyntaxError(line, column int, msg string) wdlSyntaxError {
	return wdlSyntaxError{line, column, msg}
}

type wdlErrorListener struct {
	*antlr.DiagnosticErrorListener
	syntaxErrors []wdlSyntaxError
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
		l.syntaxErrors, newWdlSyntaxError(line, column, msg),
	)
}
