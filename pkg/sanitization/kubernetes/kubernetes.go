package kubernetes

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// MetadataNameSanitizer returns a sanitized metadata name when applied.
var MetadataNameSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// replace _:/ with - and convert to lowercase
		{
			Pattern:     regexp.MustCompile(`[_:/]+`),
			Replacement: "-",
			Lowercase:   true,
		},
		// strip any leading non alpha characters
		{
			Pattern:     regexp.MustCompile(`^[^a-z0-9]+`),
			Replacement: "",
		},
		// strip any ending non alpha characters
		{
			Pattern:     regexp.MustCompile(`[^a-z0-9]$`),
			Replacement: "",
		},
		// strip any other invalid characters
		{
			Pattern:     regexp.MustCompile(`[^a-z0-9-.]+`),
			Replacement: "",
		},
	}, 253)

// HelmValueSanitizer returns a sanitized helm value when applied.
var HelmValueSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9]+`),
			Replacement: "",
		},
	}, 100)

// HelmReleaseNameSanitizer returns a sanitized helm release name when applied. currently strips periods.
var HelmReleaseNameSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9-]+`),
			Replacement: "",
			Lowercase:   true,
		},
		{
			Pattern:     regexp.MustCompile(`^-+`),
			Replacement: "",
		},
		{
			Pattern:     regexp.MustCompile(`-+$`),
			Replacement: "",
		},
	}, 53)

// RFC1123LabelSanitizer see: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names returns a sanitized helm release name when applied. currently strips periods.
var RFC1123LabelSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9-]+`),
			Replacement: "",
			Lowercase:   true,
		},
		{
			Pattern:     regexp.MustCompile(`^-+`),
			Replacement: "",
		},
		{
			Pattern:     regexp.MustCompile(`-+$`),
			Replacement: "",
		},
	}, 53)

// RFC1035LabelSanitizer see: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
var RFC1035LabelSanitizer = RFC1123LabelSanitizer

// LabelValueSanitizer returns a sanitized label value when applied.
var LabelValueSanitizer = sanitization.NewSanitizer(
	/*
		Valid label value:

		must be 63 characters or less (can be empty),
		unless empty, must begin and end with an alphanumeric character ([a-z0-9A-Z]),
		could contain dashes (-), underscores (_), dots (.), and alphanumerics between.
	*/
	[]sanitization.Rule{
		{
			// Replace all colons with underscores (for things like klotho resource ids)
			Pattern:     regexp.MustCompile(`:`),
			Replacement: "_",
		},
		{
			// Strip all invalid characters
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9-_.]+`),
			Replacement: "",
		},
		{
			// Strip leading non-alphanumeric characters
			Pattern:     regexp.MustCompile(`^[^a-zA-Z0-9]+`),
			Replacement: "",
		},
		{
			// Strip trailing non-alphanumeric characters
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9]+$`),
			Replacement: "",
		},
	}, 63)
