package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

var S3BucketSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// must not contain non-alphanumeric characters other than "-" or "."
		// uppercase characters will be converted to lowercase
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z\d.-]`),
			Replacement: "-",
			Lowercase:   true,
		},
		{
			// must not start with "xn--"
			Pattern:     regexp.MustCompile(`^(xn--)+`),
			Replacement: "",
		},
		{
			// must start and end with a letter or number
			Pattern:     regexp.MustCompile(`^[^a-z\d]+|[^a-z\d]+$`),
			Replacement: "",
		},
		{
			// must not contain repeated periods
			Pattern:     regexp.MustCompile(`\.\.+`),
			Replacement: ".",
		},
		{
			// must not be formatted as an IP address (for example, 192.168.5.4)
			Pattern:     regexp.MustCompile(`^\d{1,3}(\.)\d{1,3}(\.)\d{1,3}(\.)\d{1,3}$`),
			Replacement: "-",
		},
		{
			// must not contain hyphens adjacent to periods
			Pattern:     regexp.MustCompile(`-+\.-+`),
			Replacement: ".",
		},
		{
			// must not contain suffix "-s3alias"
			Pattern:     regexp.MustCompile(`(-s3alias)+$`),
			Replacement: "",
		},
		{
			// must not end with the suffix "--ol-s3"
			Pattern:     regexp.MustCompile(`(--ol-s3)+$`),
			Replacement: "",
		},
	},
	52, // We know that we will prepend account id onto the names here, so we want to shorten this further until we do that in the iac level
)

var S3ObjectSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`^.$`),
			Replacement: "-",
		},
	},
	0,
)
