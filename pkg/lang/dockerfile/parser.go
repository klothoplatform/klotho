package dockerfile

import (
	"io"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/smacker/go-tree-sitter/dockerfile"
)

const DockerfileLang = core.LanguageId("dockerfile")

var language = core.SourceLanguage{
	ID:               DockerfileLang,
	Sitter:           dockerfile.GetLanguage(),
	CapabilityFinder: lang.NewCapabilityFinder("comment", lang.RegexpRemovePreprocessor(`^#\s*`), lang.IsNumberCommentBlock),
}

func NewFile(path string, content io.Reader) (f *core.SourceFile, err error) {
	return core.NewSourceFile(path, content, language)
}
