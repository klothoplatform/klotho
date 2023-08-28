package main

import (
	"github.com/klothoplatform/klotho/pkg/engine"
	"github.com/spf13/cobra"
)

func main() {
	em := &engine.EngineMain{}
	var root = &cobra.Command{
		Use: "klotho",
	}
	err := em.AddEngineCli(root)
	if err != nil {
		panic(err)
	}
	err = root.Execute()
	if err != nil {
		panic(err)
	}
}
