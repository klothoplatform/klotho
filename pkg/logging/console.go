package logging

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/pborman/ansi"
	"go.uber.org/atomic"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var (
	pool = buffer.NewPool()

	levelColours = map[zapcore.Level]*color.Color{
		zapcore.DebugLevel:  color.New(color.FgMagenta),
		zapcore.InfoLevel:   color.New(color.FgHiGreen),
		zapcore.WarnLevel:   color.New(color.FgHiYellow, color.Bold),
		zapcore.ErrorLevel:  color.New(color.FgHiRed, color.Bold),
		zapcore.DPanicLevel: color.New(color.FgHiRed, color.Bold),
		zapcore.PanicLevel:  color.New(color.FgHiRed, color.Bold),
		zapcore.FatalLevel:  color.New(color.FgHiRed, color.Bold),
	}

	levelWidth int
	levelPad   string
	levelFmt   string

	annotationColour = color.New(color.FgHiCyan, color.Faint)
)

func init() {
	for l := range levelColours {
		ll := len(l.String())
		if levelWidth < ll {
			levelWidth = ll
		}
	}
	levelPad = strings.Repeat(" ", levelWidth)
	levelFmt = fmt.Sprintf("%%%ds", levelWidth)
}

type ConsoleEncoder struct {
	Verbose bool

	File        fileField
	Annotation  annotationField
	Node        astNodeField
	HadWarnings *atomic.Bool
	HadErrors   *atomic.Bool

	*bufferEncoder
}

func NewConsoleEncoder(verbose bool, hadWarnings *atomic.Bool, hadErrors *atomic.Bool) *ConsoleEncoder {
	return &ConsoleEncoder{
		Verbose:       verbose,
		HadWarnings:   hadWarnings,
		HadErrors:     hadErrors,
		bufferEncoder: &bufferEncoder{b: pool.Get()},
	}
}

func (enc *ConsoleEncoder) Clone() zapcore.Encoder {
	ne := &ConsoleEncoder{
		bufferEncoder: &bufferEncoder{b: pool.Get()},
		Verbose:       enc.Verbose,
		HadWarnings:   enc.HadWarnings,
		HadErrors:     enc.HadErrors,
		File:          enc.File,
		Annotation:    enc.Annotation,
	}
	_, _ = ne.bufferEncoder.b.Write(enc.b.Bytes())

	return ne
}

func (enc *ConsoleEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	switch obj := marshaler.(type) {
	case fileField:
		// Clone in case the file gets modified and reparsed after adding
		// we still want the content to be the old content
		enc.File = obj

	case annotationField:
		enc.Annotation = obj

	case astNodeField:
		enc.Node = obj

	default:
		return enc.bufferEncoder.AddObject(key, marshaler)
	}
	return nil
}

func (enc *ConsoleEncoder) levelPadding() string {
	if enc.Verbose {
		return levelPad
	} else {
		return ""
	}
}

func (enc *ConsoleEncoder) EncodeEntry(ent zapcore.Entry, fieldList []zapcore.Field) (*buffer.Buffer, error) {
	line := pool.Get()

	if ent.Level >= zapcore.WarnLevel {
		enc.HadWarnings.Store(true)
	}
	if ent.Level >= zapcore.ErrorLevel {
		enc.HadErrors.Store(true)
	}

	var (
		file        = enc.File.f
		annotation  = enc.Annotation.a
		nodeField   = enc.Node
		postMessage = ""
		errField    error

		indentWriter = &IndentedWriter{Indentation: enc.levelPadding(), Writer: line}
	)

	fields := pool.Get()
	_, _ = fields.Write(enc.b.Bytes())
	defer fields.Free()
	fieldCount := 0
	for _, f := range fieldList {
		switch v := f.Interface.(type) {
		case fileField:
			file = v.f
			continue

		case annotationField:
			annotation = v.a
			continue

		case astNodeField:
			nodeField = v
			continue

		case postLogMessage:
			postMessage = v.Message
			continue
		case error:
			errField = v
			continue
		}
		if fieldCount > 0 {
			fields.AppendString(", ")
		}
		fieldCount++
		f.AddTo(&bufferEncoder{b: fields})
	}

	if annotation == nil {
		annotation = &types.Annotation{}
	}

	writeFields := func() {
		if fields.Len() == 0 {
			return
		}
		size := TermSize()
		padding := size.Width - printableWidth(fields.String()) + 1
		lineLength := printableWidth(line.String()) + 1
		if padding <= lineLength {
			line.AppendByte('\n')
		} else {
			padding -= lineLength
		}
		if padding < 0 {
			padding = 0
		}
		line.AppendString(strings.Repeat(" ", padding))
		line.AppendString(fields.String())
	}

	colour := levelColours[ent.Level]
	if colour == nil {
		colour = levelColours[zapcore.PanicLevel]
	}

	if enc.Verbose {
		colour.Fprintf(line, levelFmt, ent.Level.String())
		line.AppendByte(' ')
	}

	node := nodeField.n
	if node == nil {
		node = annotation.Node
	}

	switch {
	case file != nil && node != nil:
		start := node.StartPoint()
		showDetails := enc.Verbose || ent.Level >= zapcore.WarnLevel
		if showDetails {
			// If we're going to show details, that will already include the filename and line numbers
			colour.Fprint(line, ent.Message)
		} else {
			colour.Fprintf(line, "%s:%d:%d: %s", file.Path(), start.Row, start.Column, ent.Message)
		}
		indentWriter.Indentation += colour.Sprint("| ")
		writeFields()
		if showDetails {
			if annotation.Capability != nil {
				line.AppendString("\n")
				fmt.Fprintf(&IndentedWriter{
					Indentation: indentWriter.Indentation,
					Writer:      line,
					Colour:      annotationColour,
				}, "%+v", annotation)
			}
			line.AppendString("\n")
			if ast, ok := file.(*types.SourceFile); ok {
				if node != annotation.Node {
					fmt.Fprintf(indentWriter, "in (non-annotated) %s", ast.Path())
				} else {
					fmt.Fprintf(indentWriter, "in %s", ast.Path())
				}

				nodeContent := ""
				if nodeField.n != nil {
					nodeContent = nodeField.n.Content()
				}
				if nodeContent == "" {
					nodeContent = node.Content()
				}
				line.AppendString("\n")
				fmt.Fprintf(indentWriter, "%+v", &types.NodeContent{
					Endpoints: node,
					Content:   nodeContent,
				})
				line.AppendString("\n")
			}
			if postMessage != "" {
				fmt.Fprint(indentWriter, colour.Sprint(postMessage))
			}
			line.AppendString("\n") // add an extra line for multi-line messages for readability
			return line, nil
		}
		if postMessage != "" {
			fmt.Fprint(indentWriter, colour.Sprint(postMessage))
		}
		line.AppendString("\n") // add an extra line for multi-line messages for readability

	case file != nil:
		colour.Fprintf(line, "%s: %s", file.Path(), ent.Message)
		writeFields()
		line.AppendByte('\n')
		if postMessage != "" {
			colour.Fprint(indentWriter, postMessage)
			line.AppendByte('\n')
		}

	default:
		colour.Fprintf(line, "%s", ent.Message)
		writeFields()
		line.AppendByte('\n')
		if postMessage != "" {
			colour.Fprint(indentWriter, postMessage)
			line.AppendByte('\n')
		}
	}
	if errField != nil {
		indentWriter.Indentation += colour.Sprint("| ")
		errFmt := "%v"
		if enc.Verbose {
			errFmt = "%+v"
		}
		errText := fmt.Sprintf("ERROR: "+errFmt, errField)
		for _, errLine := range strings.Split(errText, "\n") {
			fmt.Fprintf(&IndentedWriter{
				Indentation: indentWriter.Indentation,
				Writer:      line,
				Colour:      colour,
			}, "%s\n", errLine)
		}
	}
	return line, nil
}

func printableWidth(s string) (c int) {
	if s2, err := ansi.Strip([]byte(s)); err == nil {
		s = string(s2)
	}
	for _, r := range s {
		switch {
		case unicode.IsPrint(r):
			c++

		case r == '\t':
			c += 4 // assume 4-width tabs
		}
	}
	return
}
