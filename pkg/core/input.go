package core

type (
	InputFiles ConcurrentMap[string, File]
)

var InputFilesKind = "input_files"

func (*InputFiles) Key() ResourceKey {
	return ResourceKey{
		Kind: InputFilesKind,
	}
}

func (input *InputFiles) Add(f File) {
	m := (*ConcurrentMap[string, File])(input)
	if f != nil {
		m.Set(f.Path(), f)
	}
}

func (input *InputFiles) Files() map[string]File {
	m := (*ConcurrentMap[string, File])(input)
	fs := make(map[string]File)
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
