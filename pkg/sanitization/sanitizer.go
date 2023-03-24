package sanitization

import (
	"regexp"
)

type (
	Sanitizer struct {
		rules     []Rule
		maxLength int
	}

	Rule struct {
		Pattern     *regexp.Regexp
		Replacement string
	}
)

// Apply sequentially applies a Sanitizer's rules to the supplied input and returns the sanitized result.
func (s *Sanitizer) Apply(input string) string {
	output := input
	for _, rule := range s.rules {
		output = rule.Pattern.ReplaceAllString(output, rule.Replacement)
	}
	if s.maxLength != 0 && s.maxLength < len(output) {
		output = output[:s.maxLength]
	}
	return output
}

// NewSanitizer returns a new Sanitizer that applies the supplied rules to inputs.
func NewSanitizer(rules []Rule, maxLength int) *Sanitizer {
	return &Sanitizer{rules: rules, maxLength: maxLength}
}
