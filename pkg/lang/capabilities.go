package lang

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/query"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
)

type (
	commentBlock struct {
		comment       string
		endNode       *sitter.Node
		startNode     *sitter.Node
		annotatedNode *sitter.Node
	}

	capabilityFinder struct {
		sitterQuery            string
		preprocessor           CommentPreprocessor
		mergeCommentsPredicate MergeCommentsPredicate
	}

	// CommentPreprocessor edits a given comment string.
	//
	// Its input is the current comment string. Its output is what the comment string should be instead; this can just be the
	// input string, if no edits are necessary. Empty strings are *not* treated specially: they just look like an empty comment.
	CommentPreprocessor func(comment string) string

	MergeCommentsPredicate func(previous, current *sitter.Node) bool
)

// RegexpRemovePreprocessor returns a preprocessor powered by a regexp that removes all matches.
//
// The comment will be amended via `regexp.MustCompile(pattern).ReplaceAllString(comment, "")`. The preprocessor will always
// combine comment blocks.
func RegexpRemovePreprocessor(pattern string) CommentPreprocessor {
	regexp := regexp.MustCompile(pattern)
	return func(comment string) string {
		result := regexp.ReplaceAllString(comment, "")
		return result
	}
}

func CompositePreprocessor(preprocessors ...CommentPreprocessor) CommentPreprocessor {
	return func(comment string) string {
		for _, pre := range preprocessors {
			comment = pre(comment)
		}
		return comment
	}
}

func IsCLineCommentBlock(previous, current *sitter.Node) bool {
	return previous != nil && current != nil &&
		current.StartPoint().Row-previous.StartPoint().Row == 1 &&
		strings.HasPrefix(current.Content(), "//") &&
		strings.HasPrefix(previous.Content(), "//")
}

func IsHashCommentBlock(previous, current *sitter.Node) bool {
	return previous != nil && current != nil &&
		current.StartPoint().Row-previous.StartPoint().Row == 1 &&
		strings.HasPrefix(current.Content(), "#") &&
		strings.HasPrefix(previous.Content(), "#")
}

// NewCapabilityFinder creates a struct that you can use to find capabilities (annotations) within a source file.
//
// To do this, you provide a `sitterQuery` that looks for comments nodes that contain the `klotho::` annotations,
// as well as a preprocessor that runs over each of those nodes.
//
// If a source file contains multiple comment nodes in a row (as identified by having equal `.Type()`s), those comments
// will be preprocessed individually, but then merged into a single annotation.
func NewCapabilityFinder(sitterQuery string, preprocessor CommentPreprocessor, mergePredicate MergeCommentsPredicate) core.CapabilityFinder {
	return &capabilityFinder{
		sitterQuery:            sitterQuery,
		preprocessor:           preprocessor,
		mergeCommentsPredicate: mergePredicate,
	}
}

// FindAllCapabilities finds all the annotations (ie, capabilities) in a SourceFile.
func (c *capabilityFinder) FindAllCapabilities(f *core.SourceFile) (core.AnnotationMap, error) {
	var merr multierr.Error
	capabilities := make(core.AnnotationMap)
	for _, block := range c.findAllCommentBlocks(f) {
		results := annotation.ParseCapabilities(block.comment)
		for i, result := range results {
			cap := result.Capability
			err := result.Error
			var node *sitter.Node

			// Only the annotation closest to the annotated node is attached.
			// The remaining capabilities are considered to be "file-level" annotations.
			if i == len(results)-1 {
				node = block.annotatedNode
			}
			annot := &core.Annotation{Capability: cap, Node: node}
			if err != nil {
				merr.Append(core.NewCompilerError(f, annot, errors.Wrap(err, "error parsing annotation")))
				continue
			}
			capabilities.Add(annot)
		}
	}
	return capabilities, merr.ErrOrNil()
}

func (c *capabilityFinder) findAllCommentBlocks(f *core.SourceFile) []*commentBlock {
	const fullCaptureName = "fullQueryCaptureForFindAllCommentsBlocks" // please don't use this in your query ;)
	queryString := fmt.Sprintf(`(%s) @%s`, c.sitterQuery, fullCaptureName)
	nextMatch := query.Exec(f.Language, f.Tree().RootNode(), queryString)

	var blocks []*commentBlock
	for {
		match, found := nextMatch()
		if !found || match == nil {
			break
		}
		capture := match[fullCaptureName]
		comment := c.preprocessor(capture.Content())
		annotatedNode := capture.NextNamedSibling()
		if match, found := query.Exec(f.Language, annotatedNode, queryString)(); found && match[fullCaptureName] == annotatedNode {
			annotatedNode = nil
		}

		var prevBlock *commentBlock
		if len(blocks) > 0 {
			prevBlock = blocks[len(blocks)-1]
		}

		if prevBlock != nil && c.mergeCommentsPredicate(prevBlock.endNode, capture) {
			prevBlock.comment = prevBlock.comment + "\n" + comment
			prevBlock.endNode = capture
			prevBlock.annotatedNode = annotatedNode
		} else {
			blocks = append(blocks, &commentBlock{comment: comment, startNode: capture, endNode: capture, annotatedNode: annotatedNode})
		}
	}
	return blocks
}

func PrintCapabilities(caps core.AnnotationMap, out io.Writer) error {
	for _, cap := range caps {
		fmt.Fprintln(out, cap.Capability, cap.Node.Content())
	}
	return nil
}
