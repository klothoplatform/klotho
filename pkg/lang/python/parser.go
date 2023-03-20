package python

import (
	"io"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/smacker/go-tree-sitter/python"
)

const py = core.LanguageId("python")

var Language = core.SourceLanguage{
	ID:               py,
	Sitter:           python.GetLanguage(),
	CapabilityFinder: lang.NewCapabilityFinder("comment", lang.RegexpRemovePreprocessor(`^#\s*`), lang.IsNumberCommentBlock),
	ToLineComment:    lang.MakeLineCommenter("# "),
}

func NewFile(path string, content io.Reader) (f *core.SourceFile, err error) {
	return core.NewSourceFile(path, content, Language)
}
