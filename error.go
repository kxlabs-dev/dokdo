package dokdo

import "fmt"

type ParseError struct {
	File    string
	Line    int
	Message string
}

type TagNotFoundError struct {
	Target string
}

type TypeMismatchError struct {
	Field    string
	Expected string
	Got      string
}

type RequiredFieldError struct {
	Field string
}

type FileNotFoundError struct {
	Path string
}

type PathTraversalError struct {
	Path string
}

type RuntimeError struct {
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("ParseError: %s line %d\n  %s", e.File, e.Line, e.Message)
}

func (e *TagNotFoundError) Error() string {
	return fmt.Sprintf("tag not found: %s", e.Target)
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("type mismatch: field %s expected %s got %s", e.Field, e.Expected, e.Got)
}

func (e *RequiredFieldError) Error() string {
	return fmt.Sprintf("required field missing: %s", e.Field)
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("file not found: %s", e.Path)
}

func (e *PathTraversalError) Error() string {
	return fmt.Sprintf("path traversal detected: %s", e.Path)
}

func (e *RuntimeError) Error() string {
	return fmt.Sprintf("runtime error: %s", e.Message)
}

type BuildError struct {
	Message string
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("build error: %s", e.Message)
}

type InvalidParamsError struct {
	Message string
}

func (e *InvalidParamsError) Error() string {
	return fmt.Sprintf("invalid params: %s", e.Message)
}
