package collectionutil

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func Test_AppendUnique(t *testing.T) {
	cases := []struct {
		name   string
		inputs [][]int
		b      []int
		want   []int
	}{
		{
			name: "unique elems",
			inputs: [][]int{
				{1, 2, 3},
				{4, 5, 6},
			},
			want: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name: "first is non-unique",
			inputs: [][]int{
				{1, 2, 2},
				{3, 4},
			},
			want: []int{1, 2, 3, 4},
		},
		{
			name: "second is non-unique",
			inputs: [][]int{
				{1, 2},
				{3, 3, 4},
			},
			want: []int{1, 2, 3, 4},
		},
		{
			name: "both unique but share elements",
			inputs: [][]int{
				{1, 2, 3},
				{4, 3, 5},
			},
			want: []int{1, 2, 3, 4, 5},
		},
		{
			name: "one is nil",
			inputs: [][]int{
				{1, 2, 3},
				nil,
			},
			want: []int{1, 2, 3},
		},
		{
			name:   "overall is nil",
			inputs: nil,
			want:   nil,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := FlattenUnique(tt.inputs...)
			assert.Equal(tt.want, actual)
		})
	}
}

func Test_Contains(t *testing.T) {
	cases := []struct {
		name   string
		inputs []int
		b      int
		want   bool
	}{
		{
			name: "contains",
			inputs: []int{
				4, 5, 6,
			},
			b:    4,
			want: true,
		},
		{
			name: " does not contain",
			inputs: []int{
				4, 5, 6,
			},
			b:    3,
			want: false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := Contains(tt.inputs, tt.b)
			assert.Equal(tt.want, actual)
		})
	}
}
