package graphtest

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

func ParseId(t *testing.T, str string) (id construct.ResourceId) {
	err := id.UnmarshalText([]byte(str))
	if err != nil {
		t.Fatalf("failed to parse resource id %q: %v", str, err)
	}
	return
}

func ParseEdge(t *testing.T, str string) construct.Edge {
	var io construct.SimpleEdge
	err := io.UnmarshalText([]byte(str))
	if err != nil {
		t.Fatalf("failed to parse edge %q: %v", str, err)
	}
	return construct.Edge{
		Source: io.Source,
		Target: io.Target,
	}
}
