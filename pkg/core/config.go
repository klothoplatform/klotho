package core

type (
	Config struct {
		Name   string
		Secret bool
	}
)

const ConfigKind = "config"

func (p *Config) Key() ResourceKey {
	return ResourceKey{
		Name: p.Name,
		Kind: ConfigKind,
	}
}
