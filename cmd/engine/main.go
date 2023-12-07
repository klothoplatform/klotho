package main

import (
	"fmt"
	"os"

	engine "github.com/klothoplatform/klotho/pkg/engine2"
	"github.com/spf13/cobra"
)

func main() {
	root := newRootCmd()
	err := root.Execute()
	if err != nil {
		switch err.(type) {
		case engine.ConfigValidationError:
			fmt.Printf("Error: %v\n", err)
			os.Exit(2)
		default:
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func newRootCmd() *cobra.Command {
	em := &engine.EngineMain{}
	var root = &cobra.Command{}
	em.AddEngineCli(root)
	return root
}
