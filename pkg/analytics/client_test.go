package analytics

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAnalytics_Hash(t *testing.T) {
	userId := "klotho@example.com"

	cases := []struct {
		name   string
		given  any
		expect string
	}{
		{
			name:  "string",
			given: "hello",
			// printf '%s\n' 'klotho@example.com"hello"' | sha256sum
			// Note that json.Marshal always adds a newline to the end of output
			expect: "sha256:0a5261c4c416db5ecea3b865596c9c8cc6ff2a84697bbb9a689154e372b55649",
		},
		{
			name:  "number",
			given: 123,
			// printf '%s\n' 'klotho@example.com123' | sha256sum
			expect: "sha256:eb70010f1d91932a75a80f0edf4717bd39e841a174608261c68ed87bb0f73dc2",
		},
		{
			name:  "bytes",
			given: []byte{1, 2, 3},
			// printf '\x01\x02\x03' | base64  ==> AQID
			// printf '%s\n' 'klotho@example.com"AQID"' | sha256sum
			expect: "sha256:ce7bae11139f0ed51b5f7b74cb773146a542d771e235440e7e2022a0be52f892",
		},
		{
			name:  "nil",
			given: nil,
			// printf '%s\n' 'klotho@example.comnull' | sha256sum
			expect: "sha256:35f7637a859d9e720d7c9736d0d90cafe23ecddecfb977e93b2c9830f91f4ff4",
		},
		{
			name:  "jsonable object",
			given: jsonConvertable{Foo: "bar"},
			// printf '%s\n' 'klotho@example.com{"foo":"bar"}' | sha256sum
			expect: "sha256:926ed2b2760419cf62f7c2ade5c75a20a26cf4ba9d32119fd7dc045ae9333a12",
		},
		{
			name:   "not jsonable object",
			given:  func() {},
			expect: "unknown",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			analytics := &Client{
				UserId: userId,
			}
			actual := analytics.Hash(tt.given)
			assert.Equal(tt.expect, actual)
		})
	}
}

type jsonConvertable struct {
	Foo string `json:"foo"`
}
