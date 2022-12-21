package logging

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

func FileNames(files []core.File) []string {
	s := make([]string, len(files))
	for i, f := range files {
		s[i] = f.Path()
	}
	return s
}
