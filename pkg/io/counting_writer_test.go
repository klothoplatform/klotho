package io

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountingWriterTo(t *testing.T) {
	assert := assert.New(t)

	var sb strings.Builder
	counter := CountingWriter{
		Delegate: &sb,
	}

	// Write some bytes, and check the return values and aggregated BytesWritten
	n, err := counter.Write([]byte{'a', 'b'})
	assert.NoError(err)
	assert.Equal(2, n)
	assert.Equal(2, counter.BytesWritten)

	n, err = counter.Write([]byte{'c'})
	assert.NoError(err)
	assert.Equal(1, n)
	assert.Equal(3, counter.BytesWritten)

	// Check that the writes made it through to the underlying writer
	assert.Equal("abc", sb.String())
}
