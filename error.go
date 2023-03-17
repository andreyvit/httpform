package httpform

import (
	"fmt"
	"strings"
)

type Error struct {
	code    int
	message string
	cause   error
}

func (e *Error) HTTPCode() int {
	return e.code
}

func (e *Error) Message() string {
	return e.message
}

func (e *Error) Unwrap() error {
	return e.cause
}

func (e *Error) Error() string {
	var buf strings.Builder
	if e.code != 0 {
		fmt.Fprintf(&buf, "HTTP %d", e.code)
	}
	if e.message != "" {
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(e.message)
	}
	if e.cause != nil {
		if buf.Len() > 0 {
			buf.WriteString(": ")
		}
		buf.WriteString(e.cause.Error())
	}
	return buf.String()
}
