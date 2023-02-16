package core

type (
	ProxyResource struct {
		Name string
	}
)

const KlothoProxyName = "klotho_proxy"

const ProxyKind = "proxy"

func (p *ProxyResource) Key() ResourceKey {
	return ResourceKey{
		Name: p.Name,
		Kind: ProxyKind,
	}
}
