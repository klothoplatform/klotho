package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// DynamoDBTableSanitizer returns a sanitized DynamoDB Table name.
var DynamoDBTableSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{{
		Pattern:     regexp.MustCompile(`[^a-zA-Z0-9_.-]+`),
		Replacement: "_",
	}}, 255)
