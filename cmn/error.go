package cmn

import (
	"bytes"
	"fmt"
	"strings"
)

// ErrFunc returns the formatted Err
type ErrFunc func(params ...interface{}) error

// Err framework error messages pattern
func Err(code string, textAndDetails ...string) ErrFunc {
	buf := &bytes.Buffer{}
	buf.WriteByte('[')
	buf.WriteString(code)
	buf.WriteString("] ")
	buf.WriteString(textAndDetails[0])
	if !strings.HasSuffix(textAndDetails[0], ".") {
		buf.WriteByte('.')
	}

	size := len(textAndDetails)
	if size > 1 {
		buf.WriteString(" {")
		for i := 1; i < size; i++ {
			if i > 1 {
				buf.WriteString(", ")
			} else {
				buf.WriteByte(' ')
			}
			buf.WriteString(textAndDetails[i])
		}
		buf.WriteString(" }")
	}

	format := buf.String()

	return func(params ...interface{}) error {
		return fmt.Errorf(format, params...)
	}
}
