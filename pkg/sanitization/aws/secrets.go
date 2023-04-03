package aws

import (
	"github.com/klothoplatform/klotho/pkg/sanitization"
	"regexp"
)

var SecretSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^\w/+=.@-]`),
			Replacement: "-",
		},
	},
	512,
)
