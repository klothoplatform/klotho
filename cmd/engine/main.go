package main

import (
	"os"

	engine "github.com/klothoplatform/klotho/pkg/engine"
	"github.com/spf13/cobra"
)

func main() {
	root := newRootCmd()
	err := root.Execute()
	if err == nil {
		return
	}
	// Shouldn't happen, the engine CLI should handle errors
	os.Exit(1)
}

func newRootCmd() *cobra.Command {
	em := &engine.EngineMain{}
	var root = &cobra.Command{}
	em.AddEngineCli(root)
	return root
}
