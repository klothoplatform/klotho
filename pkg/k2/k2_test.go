package k2

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/klothoplatform/klotho/pkg/k2/language_host"
)

func TestK2(t *testing.T) {

	tests, err := filepath.Glob(filepath.Join("testdata", "*", "infra.py"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range tests {
		tc := testCase{inputPath: p}
		t.Run(filepath.Base(p), tc.Test)
	}
}

type testCase struct {
	inputPath string
}

func (tc testCase) Test(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var langHost language_host.LanguageHost
	err := langHost.Start(ctx, language_host.DebugConfig{})
	if err != nil {
		t.Fatalf("Failed to start language host: %v", err)
		return
	}
	defer func() {
		if err := langHost.Close(); err != nil {
			t.Fatalf("Failed to close language host: %v", err)
		}
	}()
}
