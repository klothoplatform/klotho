package main

import (
	"github.com/klothoplatform/klotho/pkg/infra"
	"github.com/spf13/cobra"
)

func main() {
	iac := &infra.IacCli{}
	var root = &cobra.Command{}
	err := iac.AddIacCli(root)
	if err != nil {
		panic(err)
	}
	err = root.Execute()
	if err != nil {
		panic(err)
	}
}
