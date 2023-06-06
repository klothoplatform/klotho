package ioutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoodWrite(t *testing.T) {
	assert := assert.New(t)

	var count int64
	var err error
	buf := strings.Builder{}

	helper := NewWriteToHelper(&buf, &count, &err)

	helper.Write("hello") // 5 chars
	assert.NoError(err)
	assert.Equal(int64(5), count)
	assert.Equal("hello", buf.String())

	helper.Write(", world") // 7 chars
	assert.NoError(err)
	assert.Equal(int64(5+7), count)
	assert.Equal("hello, world", buf.String())
}

func TestBadWrite(t *testing.T) {
	assert := assert.New(t)

	var count int64
	var err error
	buf := dummyBuffer{}

	helper := NewWriteToHelper(&buf, &count, &err)

	helper.Write("hello") // 5 chars
	assert.NoError(err)
	assert.Equal(int64(5), count)

	buf.failAfterWrite = true
	helper.Write(", world") // 7 chars
	assert.Error(err)
	assert.Equal(int64(5+7), count) // note: the writer "wrote" the 7 chars before it failed

	helper.Write(", and then some!")
	assert.Error(err)
	assert.Equal(int64(5+7), count) // note: no new elements written

}

type dummyBuffer struct {
	failAfterWrite bool
}

func (d *dummyBuffer) Write(p []byte) (n int, err error) {
	n += len(p)
	if d.failAfterWrite {
		err = fmt.Errorf("some error")
	}
	return
}
