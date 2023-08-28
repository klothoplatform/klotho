package lang

import (
	"strings"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
)

func MakeLineCommenter(commentMarker string) types.Commenter {
	return func(input string) string {
		lines := strings.Split(input, "\n")
		for i, line := range lines {
			lines[i] = commentMarker + line
		}
		return strings.Join(lines, "\n") + "\n"
	}
}
