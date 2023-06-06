package sanitization

import (
	"regexp"
)

// EnvVarKeySanitizer returns a sanitized environment key when applied.
var EnvVarKeySanitizer = NewSanitizer(
	[]Rule{
		// strip any leading non alpha characters
		{
			Pattern:     regexp.MustCompile(`^[^a-zA-Z]+`),
			Replacement: "",
		},
		// replace "-" or whitespace with "_"
		{
			Pattern:     regexp.MustCompile(`[-\s]+`),
			Replacement: "_",
		},
		// strip any other invalid characters
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9_]+`),
			Replacement: "",
		},
	}, 0)
