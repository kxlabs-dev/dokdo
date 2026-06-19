package builder

// 이 패키지의 에러 타입들은 상위 dokdo 패키지(error.go)의 동명 타입과 1:1 대응된다.
// dokdo.Build()가 errors.As()로 이 타입들을 감지해 dokdo.XxxError로 변환한다.
// 필드를 추가/변경할 경우 dokdo/error.go의 대응 타입도 함께 수정할 것.

import "fmt"

type TypeMismatchError struct {
	Field    string
	Expected string
	Got      string
}

type RequiredFieldError struct {
	Field string
}

type RuntimeError struct {
	Message string
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("type mismatch: field %s expected %s got %s", e.Field, e.Expected, e.Got)
}

func (e *RequiredFieldError) Error() string {
	return fmt.Sprintf("required field missing: %s", e.Field)
}

func (e *RuntimeError) Error() string {
	return fmt.Sprintf("runtime error: %s", e.Message)
}
