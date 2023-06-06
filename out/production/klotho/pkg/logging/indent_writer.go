package logging

import (
	"fmt"
	"io"
	"regexp"

	"github.com/fatih/color"
)

type IndentedWriter struct {
	Indentation string
	Writer      io.Writer
	Colour      *color.Color
}

var lineIndentRE = regexp.MustCompile(`(?m)^`)

func (w *IndentedWriter) Write(b []byte) (n int, err error) {
	lines := lineIndentRE.Split(string(b), -1)
	for _, line := range lines {
		if w.Colour != nil {
			line = w.Colour.Sprint(line)
		}
		m, err := fmt.Fprintf(w.Writer, "%s%s", w.Indentation, line)
		if err != nil {
			return n, err
		}
		n += m
	}
	return
}
