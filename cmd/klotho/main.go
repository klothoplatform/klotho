package main

import (
	"github.com/klothoplatform/klotho/pkg/cli"
)

func main() {
	km := cli.KlothoMain{
		DefaultUpdateStream: "oss:latest",
		Version:             Version,
		PluginSetup: func(psb *cli.PluginSetBuilder) error {
			return psb.AddAll()
		},
	}
	km.Main()
}
