package aws

import (
	"github.com/klothoplatform/klotho/pkg/sanitization"
	"regexp"
)

// EcsTaskDefinitionSanitizer returns a sanitized ECS TaskDefinition name when applied.
var EcsTaskDefinitionSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// strip any characters not matching [a-zA-Z0-9-_]
		{
			Pattern:     regexp.MustCompile(`[^\w-]+`),
			Replacement: "",
		},
	}, 255)

// EcsClusterSanitizer returns a sanitized ECS Cluster name when applied.
var EcsClusterSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// strip any characters not matching [a-zA-Z0-9-_]
		{
			Pattern:     regexp.MustCompile(`[^\w-]+`),
			Replacement: "",
		},
	}, 255)

// EcsServiceSanitizer returns a sanitized ECS Service name when applied.
var EcsServiceSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// strip any characters not matching [a-zA-Z0-9-_]
		{
			Pattern:     regexp.MustCompile(`[^\w-]+`),
			Replacement: "",
		},
	}, 255)
