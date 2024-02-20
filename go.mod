module github.com/klothoplatform/klotho

go 1.22.0

require (
	github.com/Masterminds/sprig/v3 v3.2.3
	github.com/alitto/pond v1.8.3
	github.com/coreos/go-semver v0.3.0
	github.com/dominikbraun/graph v0.23.0
	github.com/fatih/color v1.13.0
	github.com/gojek/heimdall/v7 v7.0.2
	github.com/google/pprof v0.0.0-20210226084205-cbba55b83ad5
	github.com/google/uuid v1.3.0
	github.com/iancoleman/strcase v0.3.0
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/lithammer/dedent v1.1.0
	github.com/pelletier/go-toml/v2 v2.0.8-0.20230509155657-d34104d49374
	github.com/pkg/errors v0.9.1
	github.com/r3labs/diff v1.1.0
	github.com/schollz/progressbar/v3 v3.13.0
	github.com/smacker/go-tree-sitter v0.0.0-20220209044044-0d3022e933c3
	github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify v1.8.2
	github.com/vmware-labs/yaml-jsonpath v0.3.2
	go.uber.org/mock v0.4.0
	go.uber.org/zap v1.22.0
	gopkg.in/yaml.v3 v3.0.1
	helm.sh/helm/v3 v3.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dprotaso/go-yit v0.0.0-20191028211022-135eb7262960 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.23.0 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/goleak v1.2.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace (
	github.com/dominikbraun/graph => github.com/klothoplatform/graph v0.24.7

	// github.com/dominikbraun/graph => github.com/klothoplatform/graph v0.24.3
	github.com/smacker/go-tree-sitter => github.com/klothoplatform/go-tree-sitter v0.1.1

	// yaml fork is the same (as of 2023/07/28) except with the PR merged:
	// https://github.com/go-yaml/yaml/pull/961
	gopkg.in/yaml.v3 => github.com/klothoplatform/yaml/v3 v3.0.1
)

require (
	github.com/Code-Hex/dd v1.1.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.0 // indirect
	github.com/alecthomas/kong v0.8.1 // indirect
	github.com/coreos/go-oidc/v3 v3.9.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.1 // indirect
	github.com/gojek/valkyrie v0.0.0-20180215180059-6aee720afcdf // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/huandu/xstrings v1.3.3 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.3.4 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/thessem/zap-prettyconsole v0.3.0 // indirect
	go.lsp.dev/jsonrpc2 v0.10.0 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	go.lsp.dev/protocol v0.12.0 // indirect
	go.lsp.dev/uri v0.3.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/exp v0.0.0-20240213143201-ec583247a57a // indirect
	golang.org/x/mod v0.15.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/oauth2 v0.13.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/term v0.17.0 // indirect
	golang.org/x/tools v0.18.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
