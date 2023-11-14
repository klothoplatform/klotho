package main

import (
	"fmt"
	"os"

	engine "github.com/klothoplatform/klotho/pkg/engine2"
	"github.com/spf13/cobra"
)

func main() {
	em := &engine.EngineMain{}
	var root = &cobra.Command{}
	em.AddEngineCli(root)
	err := root.Execute()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
