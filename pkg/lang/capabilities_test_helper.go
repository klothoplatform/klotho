package lang

import (
	"errors"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	FindAllCommentBlocksExpected struct {
		Comment string
		Node    string
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
	blocks := capFinder.findAllCommentsBlocks(f)
	found := []FindAllCommentBlocksExpected{}
	for _, block := range blocks {
		found = append(found, FindAllCommentBlocksExpected{
			Comment: block.comment,
			Node:    block.node.Content([]byte(source))})
	}
	return found, nil

}
