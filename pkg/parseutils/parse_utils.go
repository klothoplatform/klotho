package parseutils

import (
	"fmt"
	"regexp"
)

// ExpressionExtractor returns a function that returns up to n balanced expressions for the supplied start and end delimiters.
//
// The escape argument is used for detecting escaped delimiters and should typically be either `\` or `\\`
// depending on the format of the input string.
func ExpressionExtractor(escape string, start, end rune) func(input string, n int) []string {
	return func(input string, n int) []string {
		escapedStartPattern := regexp.MustCompile(fmt.Sprintf(`^[^%c]*?((?:%s)*)\%c`, start, escape, start))
		escapedEndPattern := regexp.MustCompile(fmt.Sprintf(`^[^%c]*?((?:%s)*)\%c`, end, escape, end))
		sCount := 0
		eCount := 0
		exprStartIndex := -1
		lastMatchIndex := -1
		var expressions []string
		for i := 0; i < len(input); i++ {
			switch rune(input[i]) {
			case start:
				match := escapedStartPattern.FindStringSubmatch(input[lastMatchIndex+1:])
				if match[1] == "" || len(match[1])%len(escape) != 0 {
					sCount++
				}
				lastMatchIndex = i
				if exprStartIndex < 0 {
					exprStartIndex = i
				}
			case end:
				match := escapedEndPattern.FindStringSubmatch(input[lastMatchIndex+1:])
				if match[1] == "" || len(match[1])%len(escape) != 0 {
					eCount++
				}
				lastMatchIndex = i
			}
			if sCount > 0 && sCount == eCount && exprStartIndex >= 0 {
				expressions = append(expressions, input[exprStartIndex:i+1])
				if n > 0 && len(expressions) == n {
					return expressions
				}
				// reset counters for next expression
				exprStartIndex = -1
				sCount = 0
				eCount = 0
			}
		}
		return expressions
	}
}
