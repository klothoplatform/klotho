package main

import (
	"github.com/klothoplatform/klotho/pkg/auth"
	"github.com/klothoplatform/klotho/pkg/cli"
	"github.com/spf13/pflag"
)

func main() {
	authRequirement := LocalAuth(false)
	km := cli.KlothoMain{
		DefaultUpdateStream: "open:latest",
		Version:             Version,
		PluginSetup: func(psb *cli.PluginSetBuilder) error {
			return psb.AddAll()
		},
		Authorizer: &authRequirement,
	}

	km.Main()
}

// LocalAuth is an auth.Authorizer that requires login unless its value is true.
type LocalAuth bool

func (local *LocalAuth) SetUpCliFlags(flags *pflag.FlagSet) {
	flags.BoolVar((*bool)(local), "local", bool(*local), "If provided, runs Klotho with a local login (that is, not requiring an authenticated login)")
}

func (local *LocalAuth) Authorize() error {
	if !*local {
		return auth.Authorize()
	}
	return nil
}
