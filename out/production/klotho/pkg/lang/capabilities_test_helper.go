package lang

import (
	"errors"
	"github.com/klothoplatform/klotho/pkg/query"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	FindAllCommentBlocksExpected struct {
		Comment       string
		AnnotatedNode string
	}

	FindAllCommentBlocksTestCase struct {
		Name   string
		Source string
		Want   []FindAllCommentBlocksExpected
	}

	TestRunner interface {
		Run()
	}
)

func FindAllCommentBlocksForTest(language core.SourceLanguage, source string) ([]FindAllCommentBlocksExpected, error) {
	capFinder, ok := language.CapabilityFinder.(*capabilityFinder)
	if !ok {
		return nil, errors.New("capability wasn't created with lang.NewCapabilityFinder")
	}
	f, err := core.NewSourceFile("test.js", strings.NewReader(source), language)
	if err != nil {
		return nil, err
	}
	blocks := capFinder.findAllCommentBlocks(f)
	found := []FindAllCommentBlocksExpected{}
	for _, block := range blocks {
		content := ""
		if block.endNode != nil {
			content = query.NodeContentOrEmpty(block.annotatedNode)
		}
		found = append(found, FindAllCommentBlocksExpected{
			Comment:       block.comment,
			AnnotatedNode: content,
		})
	}
	return found, nil

}
