package main

import (
	"github.com/klothoplatform/klotho/pkg/infra"
	"github.com/spf13/cobra"
)

func main() {
	var root = &cobra.Command{}
	// iac := &infra.IacCli{}
	// err := iac.AddIacCli(root)
	err := infra.AddIacCli(root)
	if err != nil {
		panic(err)
	}
	err = root.Execute()
	if err != nil {
		panic(err)
	}
}
