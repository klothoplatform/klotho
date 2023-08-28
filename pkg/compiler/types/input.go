package types

import (
	"github.com/klothoplatform/klotho/pkg/async"
	"github.com/klothoplatform/klotho/pkg/io"
)

type (
	InputFiles async.ConcurrentMap[string, io.File]
)

var InputFilesKind = "input_files"

func (input *InputFiles) Add(f io.File) {
	m := (*async.ConcurrentMap[string, io.File])(input)
	if f != nil {
		m.Set(f.Path(), f)
	}
}

func (input *InputFiles) Files() map[string]io.File {
	m := (*async.ConcurrentMap[string, io.File])(input)
	fs := make(map[string]io.File)
	for _, f := range m.Values() {
		fs[f.Path()] = f
	}
	return fs
}

func (input *InputFiles) FilesOfLang(lang LanguageId) []*SourceFile {
	var filteredFiles []*SourceFile
	for _, file := range input.Files() {
		if src, ok := lang.CastFile(file); ok {
			filteredFiles = append(filteredFiles, src)
		}
	}
	return filteredFiles
}
