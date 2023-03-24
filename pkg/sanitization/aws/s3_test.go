package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeS3BucketName(t *testing.T) {
	cases := []struct {
		rule  string
		input string
		wants string
	}{
		{
			rule:  `must consist only of lowercase letters numbers dots and hyphens`,
			input: `hello_world#hashtag?`,
			wants: `hello-world-hashtag--payloads`,
		},
		{
			rule:  `we recommend that you avoid using dots`,
			input: `hello.world`,
			wants: `hello-world-payloads`,
		},
		{
			rule:  `must not contain two adjacent periods`,
			input: `foo......bar`,
			wants: `foo------bar-payloads`,
		},
		{
			rule:  `must be between 3 and 63 chars`,
			input: `abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789`,
			// 26 letters + 10 digits + a-f = 42. Add "-payloads" => 51. Add 12-digit AWS account id => 63
			wants: `abcdefghijklmnopqrstuvwxyz0123456789abcdef-payloads`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.rule, func(t *testing.T) {
			assert := assert.New(t)
			actual := SanitizeS3BucketName(tt.input)
			assert.Equal(tt.wants, actual)
		})

	}
}
