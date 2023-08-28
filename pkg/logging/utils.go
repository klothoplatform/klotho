package logging

import klotho_io "github.com/klothoplatform/klotho/pkg/io"

func FileNames(files []klotho_io.File) []string {
	s := make([]string, len(files))
	for i, f := range files {
		s[i] = f.Path()
	}
	return s
}
