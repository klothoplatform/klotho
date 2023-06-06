package javascript

import (
	"io"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/smacker/go-tree-sitter/javascript"
)

var multilineCommentMarginRegexp = regexp.MustCompile(`(?m)^\s*[*]*[ \t]*`) // we need to use [ \t] instead of \s, because \s includes newlines in (?m) mode.

const js = core.LanguageId("javascript")

var Language = core.SourceLanguage{
	ID:     js,
	Sitter: javascript.GetLanguage(),
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
			//     * bar
			//     */
			//
			// First, we'll trim the opening and closing delimiters, to get it to:
			//   * foo
			//   * bar
			//
			comment = comment[2 : len(comment)-2]
			// Then, we'll use a regexp to remove an opening stretch of `*`s from each line
			comment = multilineCommentMarginRegexp.ReplaceAllString(comment, "")
			// `/*`-style comments never combine with subsequent comments
			return comment
		},
	),
		lang.IsCLineCommentBlock),
	ToLineComment: lang.MakeLineCommenter("// "),
}

func NewFile(path string, content io.Reader) (f *core.SourceFile, err error) {
	return core.NewSourceFile(path, content, Language)
}
