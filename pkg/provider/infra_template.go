package provider

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
)

type (
	TemplateConfig struct {
		Datadog bool
		Lumigo  bool
		AppName string
	}

	TemplateData struct {
		ConfigPath string

		HasKV       bool
		Secrets     []string
		ORMs        []ORM
		Redis       []Redis
		PubSubs     []PubSub
		ExecUnits   []ExecUnit
		HelmCharts  []HelmChart
		StaticUnits []StaticUnit
		Gateways    []Gateway
		Topology    core.TopologyData
		Results     *core.CompilationResult
	}

	PubSub struct {
		Publishers  []core.ResourceKey
		Subscribers []core.ResourceKey
		Path        string
		EventName   string
		Name        string
		Params      config.InfraParams
	}

	ORM struct {
		Name   string
		Type   string
		Params config.InfraParams
	}

	Redis struct {
		Name   string
		Type   string
		Params config.InfraParams
	}

	ExecUnit struct {
		Name                 string
		Type                 string
		MemReqMB             int
		KeepWarm             bool
		Schedules            []Schedule
		HelmOptions          config.HelmChartOptions
		Params               config.InfraParams
		EnvironmentVariables []core.EnvironmentVariable
	}

	HelmChart struct {
		Name      string
		Directory string
		Values    []kubernetes.Value
	}

	StaticUnit struct {
		Name                   string
		Type                   string
		IndexDocument          string
		ContentDeliveryNetwork config.ContentDeliveryNetwork
	}

	Gateway struct {
		Name    string
		Routes  []Route
		Targets map[string]core.GatewayTarget
	}

	Route struct {
		ExecUnitName string `json:"execUnitName"`
		Path         string `json:"path"`
		Verb         string `json:"verb"`
	}

	Schedule struct {
		ModulePath string
		FuncName   string
		Cron       string
	}
)
