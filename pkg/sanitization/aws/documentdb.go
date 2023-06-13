package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

var DocumentDbClusterSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`^([a-zA-Z])`),
			Replacement: "$1",
		},
	},
	63)

var DocumentDbInstanceSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// TODO?
	},
	63)
