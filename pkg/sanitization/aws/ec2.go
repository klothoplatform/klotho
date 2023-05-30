package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// Ec2InstanceSanitizer returns a sanitized EC2 instance name when applied.
var Ec2InstanceSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z\d-]`),
			Replacement: "_",
		},
		{
			Pattern:     regexp.MustCompile(`^[^a-zA-Z]+`),
			Replacement: "",
		},
	},
	100,
)
