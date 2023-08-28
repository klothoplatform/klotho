package yaml

import (
	"io"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/smacker/go-tree-sitter/yaml"
)

const YamlLang = types.LanguageId("yaml")

var language = types.SourceLanguage{
	ID:               YamlLang,
	Sitter:           yaml.GetLanguage(),
	CapabilityFinder: lang.NewCapabilityFinder("comment", lang.RegexpRemovePreprocessor(`^#\s*`), lang.IsHashCommentBlock),
}

func NewFile(path string, content io.Reader) (f *types.SourceFile, err error) {
	return types.NewSourceFile(path, content, language)
}
