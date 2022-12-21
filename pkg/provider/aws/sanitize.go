package aws

import (
	"regexp"
	"strings"
)

// SanitizeS3BucketName returns a valid S3 bucket name for a given app name. In addition to any sanitization, this will
// append a suffix of "-payloads". When we actually use these bucket names, we'll prefix them with the 12-digit AWS
// account id; this method assumes that for its checks.
//
// The rules we're checking for ar at https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html.
//
// Some of those (especially the ones to do with prefixes and suffixes) we get for free because of the AWS account id
// and "-payloads" suffix. Besides those, we also do not check for the following:
//
//   - "Bucket names must be unique across all AWS accounts in all the AWS Regions within a partition": it's impossible
//     to check this without connecting to AWS, which we don't want to here.
//   - "A bucket name cannot be used by another AWS account in the same partition until the bucket is deleted": isn't
//     this the same as the previous rule?
func SanitizeS3BucketName(appName string) string {
	const s3BucketNameSuffix = "-payloads"
	name := appName

	// A couple rules we get for free because of the "-payloads" suffix:
	//
	// - Bucket names must not be formatted as an IP address (for example, 192.168.5.4)
	// - Bucket names must not end with the suffix -s3alias.
	// - Bucket names must begin and end with a letter or number (the suffix gives us the end)
	//
	// When we actually create these buckets, they'll be prefixed with the 12-digit AWS account ids. That gives us a few
	// more rules for free:
	// - Bucket names must not start with the prefix xn--.
	// - Bucket names must begin and end with a letter or number (the account id gives us the beginning)

	// Bucket names can consist only of lowercase letters, numbers, dots (.), and hyphens (-).
	name = regexp.MustCompile(`[^a-z0-9.-]`).ReplaceAllLiteralString(name, "-")

	// For best compatibility, we recommend that you avoid using dots (.) in bucket names, except for buckets that are
	//used only for static website hosting.
	//
	// This isn't a hard requirement, but let's do it anyway.
	// This also addresses:
	//
	// Bucket names must not contain two adjacent periods.
	name = strings.ReplaceAll(name, ".", "-")

	// Bucket names must be between 3 (min) and 63 (max) characters long.
	//
	// The "-payloads" and AWS account id already gives us the minimum, so just check max.
	maxNameChars := 63 - len(s3BucketNameSuffix) - 12 // 12 for AWS account id
	if len(name) > maxNameChars {
		name = name[:maxNameChars]
	}

	return name + s3BucketNameSuffix

}
