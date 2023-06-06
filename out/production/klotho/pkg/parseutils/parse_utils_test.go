package parseutils

import (
	"reflect"
	"testing"
)

func TestExpressionExtractor(t *testing.T) {
	tests := []struct {
		name   string
		escape string
		start  rune
		end    rune
		input  string
		want   []string
		limit  int
	}{
		{
			name:  "extracts outermost expression",
			start: '{',
			end:   '}',
			input: "prefix{{hello}{world}}suffix",
			want:  []string{"{{hello}{world}}"},
		},
		{
			name:  "works whichever start and end runes are are provided",
			start: '(',
			end:   ')',
			input: "prefix((he{}llo)(world))suffix",
			want:  []string{"((he{}llo)(world))"},
		},
		{
			name:  "extracts multiple independent expressions if present in input",
			start: '{',
			end:   '}',
			input: "prefix{expr1}-{expr2}suffix",
			want:  []string{"{expr1}", "{expr2}"},
		},
		{
			name:  "extracts up to n expressions when the supplied limit > 0",
			start: '{',
			end:   '}',
			input: "prefix{expr1}{expr2}suffix",
			limit: 1,
			want:  []string{"{expr1}"},
		},
		{
			name:  "treats a limit < 1 as unlimited",
			start: '{',
			end:   '}',
			input: "prefix{expr1}{expr2}suffix",
			limit: 0,
			want:  []string{"{expr1}", "{expr2}"},
		},
		{
			name:   "ignores escaped start and end delimiters",
			start:  '{',
			end:    '}',
			escape: `\\`,
			input:  `prefix{expr1\\}\\{expr2}suffix`,
			want:   []string{`{expr1\\}\\{expr2}`},
		},
		{
			name:   "handles escaped escape sequences",
			start:  '{',
			end:    '}',
			escape: `\\`,
			input:  `{escaped\\\}{repeated\\\\\\{}`,
			want:   []string{`{escaped\\\}`, `{repeated\\\\\\{}`},
		},
		{
			name:   "unbalanced expression due to escape is ignored",
			start:  '{',
			end:    '}',
			escape: `\\`,
			input:  `prefix{expr1\\}{expr2}suffix`,
			want:   nil,
		},
		{
			name:   "unbalanced expression is ignored",
			start:  '{',
			end:    '}',
			escape: ``,
			input:  `{expr1{expr2}`,
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExpressionExtractor(tt.escape, tt.start, tt.end)(tt.input, tt.limit); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExpressionExtractor() = %v, want %v", got, tt.want)
			}
		})
	}
}
