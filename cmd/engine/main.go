package main

import (
	engine "github.com/klothoplatform/klotho/pkg/engine2"
	"github.com/spf13/cobra"
)

func main() {
	em := &engine.EngineMain{}
	var root = &cobra.Command{}
	err := em.AddEngineCli(root)
	if err != nil {
		panic(err)
	}
	err = root.Execute()
	if err != nil {
		panic(err)
	}
}
