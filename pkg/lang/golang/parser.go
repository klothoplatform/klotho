package golang

import (
	"io"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/smacker/go-tree-sitter/golang"
)

const goLang = types.LanguageId("go")

var multilineCommentMarginRegexp = regexp.MustCompile(`(?m)^\s*[*]*[ \t]*`) // we need to use [ \t] instead of \s, because \s includes newlines in (?m) mode.

var Language = types.SourceLanguage{
	ID:     goLang,
	Sitter: golang.GetLanguage(),
	CapabilityFinder: lang.NewCapabilityFinder("comment", lang.CompositePreprocessor(
		lang.RegexpRemovePreprocessor(`//\s*`),
		func(comment string) string {
			// Check for comments starting with `/*`.
			// If you don't find one, just return this comment unchanged.
			// If you do find one, snip off the start and end chars, as well as any `*`s that prefix a line
			// (this is a common style for giving the comment a left border).
			if !strings.HasPrefix(comment, "/*") {
				return comment
			}
			// The comment is something like:
			//   /** foo
			//    * bar
			//    */
			//
			// First, we'll trim the opening and closing slashes, to get it to:
			//   ** foo
			//    * bar
			//    *
			//
			// Then, we'll use a regexp to remove an opening stretch of `*`s from each line
			comment = comment[1 : len(comment)-1]
			comment = multilineCommentMarginRegexp.ReplaceAllString(comment, "")
			// `/*`-style comments never combine with subsequent comments
			return comment
		},
	),
		lang.IsCLineCommentBlock),
}

func NewFile(path string, content io.Reader) (f *types.SourceFile, err error) {
	return types.NewSourceFile(path, content, Language)
}
