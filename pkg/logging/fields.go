package logging

import (
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/core"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type fileField struct {
	f core.File
}

func (field fileField) Sanitize(hasher func(any) string) SanitizedField {
	extension := "unknown"
	switch ext := filepath.Ext(field.f.Path()); ext {
	case ".json", ".js", ".ts", ".yaml":
		extension = ext
	}
	return SanitizedField{
		Key: "FileExtension",
		Content: map[string]any{
			"extension": extension,
			"path":      hasher(field.f.Path()),
		},
	}
}

func (field fileField) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("path", field.f.Path())
	return nil
}

func FileField(f core.File) zap.Field {
	return zap.Object("file", fileField{f: f.Clone()})
}

type annotationField struct {
	a *core.Annotation
}

func (field annotationField) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	_ = astNodeField{n: field.a.Node}.MarshalLogObject(enc)
	enc.AddString("capability", field.a.Capability.Name)
	return nil
}

func (field annotationField) Sanitize(hasher func(any) string) SanitizedField {
	return SanitizedField{
		Key: "Capability",
		Content: map[string]any{
			"name":       field.a.Capability.Name,
			"id":         hasher(field.a.Capability.ID),
			"directives": hasher(field.a.Capability.Directives),
		},
	}
}

func AnnotationField(a *core.Annotation) zap.Field {
	return zap.Object("annotation", annotationField{a: a})
}

type astNodeField struct {
	n       *sitter.Node
	content string
}

// DescribeKlothoFields is intended for unit testing expected log lines.
//
// This returns a map whose keys are the field keys, and whose values are descriptions of the Klotho-provided zap fields.
// Don't try to parse these.
//
// If any of the expected fields are missing, their values will be text saying that the field is missing.
func DescribeKlothoFields(fields []zapcore.Field, expected ...string) map[string]string {
	all := map[string]string{}

	for _, expect := range expected {
		all[expect] = "!!(MISSING)!!"
	}

	bufPool := buffer.NewPool()
	encoder := bufferEncoder{b: bufPool.Get()}
	defer encoder.b.Free()

	for _, field := range fields {
		encoder.b.Reset()
		marhaledField, ok := field.Interface.(zapcore.ObjectMarshaler)
		if !ok {
			continue
		}
		if err := encoder.AppendObject(marhaledField); err != nil {
			all[field.Key] = "!!(UNMARSHALING ERROR)"
		} else {
			all[field.Key] = encoder.b.String()
		}
	}
	return all
}

func (field astNodeField) Sanitize(hasher func(any) string) SanitizedField {
	return SanitizedField{
		Key: "AstNodeType",
		Content: map[string]any{
			"type":    field.n.Type(),
			"content": hasher(field.content),
		},
	}
}

func (field astNodeField) MarshalLogObject(enc zapcore.ObjectEncoder) error {

	start := field.n.StartPoint()
	end := field.n.EndPoint()

	enc.AddUint32("start-row", start.Row)
	enc.AddUint32("start-column", start.Column)
	enc.AddUint32("end-row", end.Row)
	enc.AddUint32("end-column", end.Column)
	return nil
}

func NodeField(n *sitter.Node, source []byte) zap.Field {
	return zap.Object("node", astNodeField{
		n:       n,
		content: n.Content(source)},
	)
}

type postLogMessage struct {
	Message string
}

func (field postLogMessage) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("post-msg", field.Message)
	return nil
}

func PostLogMessageField(msg string) zap.Field {
	return zap.Inline(postLogMessage{Message: msg})
}
