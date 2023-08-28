package python

import (
	"io"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/smacker/go-tree-sitter/python"
)

const py = types.LanguageId("python")

var Language = types.SourceLanguage{
	ID:               py,
	Sitter:           python.GetLanguage(),
	CapabilityFinder: lang.NewCapabilityFinder("comment", lang.RegexpRemovePreprocessor(`^#\s*`), lang.IsHashCommentBlock),
	ToLineComment:    lang.MakeLineCommenter("# "),
}

func NewFile(path string, content io.Reader) (f *types.SourceFile, err error) {
	return types.NewSourceFile(path, content, Language)
}
