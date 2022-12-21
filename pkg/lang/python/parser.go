package python

import (
	"io"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/smacker/go-tree-sitter/python"
)

var Language = core.SourceLanguage{
	ID:               core.LanguageId("python"),
	Sitter:           python.GetLanguage(),
	CapabilityFinder: lang.NewCapabilityFinder("comment", lang.RegexpRemovePreprocessor(`^#\s*`)),
	TurnIntoComment:  lang.MakeLineCommenter("# "),
}

func NewFile(path string, content io.Reader) (f *core.SourceFile, err error) {
	return core.NewSourceFile(path, content, Language)
}
