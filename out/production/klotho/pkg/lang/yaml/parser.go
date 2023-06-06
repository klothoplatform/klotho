package yaml

import (
	"io"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/smacker/go-tree-sitter/yaml"
)

const YamlLang = core.LanguageId("yaml")

var language = core.SourceLanguage{
	ID:               YamlLang,
	Sitter:           yaml.GetLanguage(),
	CapabilityFinder: lang.NewCapabilityFinder("comment", lang.RegexpRemovePreprocessor(`^#\s*`), lang.IsHashCommentBlock),
}

func NewFile(path string, content io.Reader) (f *core.SourceFile, err error) {
	return core.NewSourceFile(path, content, language)
}
