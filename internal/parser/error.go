package parser

import "fmt"

type ParseError struct {
	File    string
	Line    int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("ParseError: %s line %d\n  %s", e.File, e.Line, e.Message)
}
