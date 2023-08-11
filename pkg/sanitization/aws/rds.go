package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// RdsInstanceSanitizer returns a sanitized rds instance name when applied.
var RdsInstanceSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^\da-z-]`),
			Replacement: "-",
		},
		// Identifier must start with a letter
		{
			Pattern:     regexp.MustCompile(`^[^a-zA-Z]+`),
			Replacement: "",
		},
		// Identifier must not end with a hyphen
		{
			Pattern:     regexp.MustCompile(`-+$`),
			Replacement: "",
		},
		// Identifier must not contain consecutive hyphens
		{
			Pattern:     regexp.MustCompile(`--+`),
			Replacement: "-",
		},
	}, 64)

// RdsSubnetGroupSanitizer returns a sanitized subnet group name when applied.
var RdsSubnetGroupSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-z0-9_.-]+`),
			Replacement: "",
		},
	}, 64)

// RdsProxySanitizer returns a sanitized proxy name when applied.
var RdsProxySanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-z0-9-]`),
			Replacement: "",
		},
	}, 64)

var RdsDBNameSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// Identifier must contain only alphanumeric characters
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9]+`),
			Replacement: "",
		},
		// Identifier must start with a letter
		{
			Pattern:     regexp.MustCompile(`^[^a-zA-Z]+`),
			Replacement: "",
		},
	}, 128)
