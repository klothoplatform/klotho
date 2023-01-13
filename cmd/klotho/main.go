package main

import (
	"github.com/klothoplatform/klotho/pkg/cli"
	"github.com/klothoplatform/klotho/pkg/updater"
)

func main() {
	km := cli.KlothoMain{
		UpdateStream: updater.DefaultStream,
		Version:      Version,
		PluginSetup: func(psb *cli.PluginSetBuilder) error {
			return psb.AddAll()
		},
	}
	km.Main()
}
