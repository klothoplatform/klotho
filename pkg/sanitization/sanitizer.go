package sanitization

import "regexp"

type (
	Sanitizer struct {
		rules []Rule
	}

	Rule struct {
		Pattern     *regexp.Regexp
		Replacement string
	}
)

func (s *Sanitizer) Apply(input string) string {
	output := input
	for _, rule := range s.rules {
		output = rule.Pattern.ReplaceAllString(output, rule.Replacement)
	}
	return output
}

func NewSanitizer(rules ...Rule) *Sanitizer {
	return &Sanitizer{rules: rules}
}
