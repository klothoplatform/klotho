package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func TestInfraTemplateModification(t *testing.T) {
	cases := []struct {
		name         string
		results      []core.CloudResource
		dependencies []core.Dependency
		cfg          config.Application
		data         TemplateData
	}{
		{
			name: "simple test",
			results: []core.CloudResource{&core.Gateway{
				Name:   "gw",
				GWType: core.GatewayKind,
				Routes: []core.Route{{Path: "/"}},
			},
				&core.ExecutionUnit{
					Name:     "unit",
					ExecType: eks,
				},
			},
			cfg: config.Application{
				Provider: "aws",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"unit": {Type: eks},
				},
			},
			dependencies: []core.Dependency{},
			data: TemplateData{
				TemplateData: provider.TemplateData{
					Gateways: []provider.Gateway{
						{Name: "gw", Routes: []provider.Route{{ExecUnitName: "", Path: "/", Verb: ""}}, Targets: map[string]core.GatewayTarget(nil)},
					},
					ExecUnits: []provider.ExecUnit{
						{Name: "unit", Type: "eks", NetworkPlacement: "private", MemReqMB: 0, KeepWarm: false, Schedules: []provider.Schedule(nil), Params: config.InfraParams{}},
					},
				},
				UseVPC: true,
			},
		},
		{
			name: "helm chart test",
			results: []core.CloudResource{
				&core.ExecutionUnit{
					Name:     "unit",
					ExecType: eks,
				},
				&kubernetes.KlothoHelmChart{Values: []kubernetes.Value{{
					Type: string(kubernetes.ImageTransformation),
					Key:  kubernetes.GenerateImagePlaceholder("unit"),
				}}},
			},
			cfg: config.Application{
				Provider: "aws",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"unit": {Type: eks, HelmChartOptions: &config.HelmChartOptions{Install: true}, NetworkPlacement: "public"},
				},
			},
			dependencies: []core.Dependency{},
			data: TemplateData{
				TemplateData: provider.TemplateData{
					ExecUnits: []provider.ExecUnit{
						{Name: "unit", Type: "eks", NetworkPlacement: "public", MemReqMB: 0, KeepWarm: false, Schedules: []provider.Schedule(nil),
							Params: config.InfraParams{}, HelmOptions: config.HelmChartOptions{Install: true}},
					},
				},
				UseVPC: true,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result := core.CompilationResult{}

			result.AddAll(tt.results)

			deps := core.Dependencies{}
			for _, dep := range tt.dependencies {
				deps.Add(dep.Source, dep.Target)
			}

			aws := AWS{
				Config: &tt.cfg,
			}

			// If we want the BuildImage to be true, we will add a dockerfile. This is because of how we initialize concurrent maps
			for _, unit := range tt.data.ExecUnits {
				res := result.Get(core.ResourceKey{Kind: core.ExecutionUnitKind, Name: unit.Name})
				resUnit, ok := res.(*core.ExecutionUnit)
				if !assert.True(ok) {
					return
				}
				resUnit.Add(&core.FileRef{FPath: "Dockerfile"})
			}

			err := aws.Transform(&result, &deps)

			if !assert.NoError(err) {
				return
			}
			awsResult := result.GetResourcesOfType(AwsTemplateDataKind)
			if !assert.Len(awsResult, 1) {
				return
			}
			data := awsResult[0]
			awsData, ok := data.(*TemplateData)
			if !assert.True(ok) {
				return
			}
			assert.Equal(tt.data.ExecUnits, awsData.ExecUnits)
			assert.Equal(tt.data.Gateways, awsData.Gateways)
			assert.Equal(tt.data.UseVPC, awsData.UseVPC)
		})
	}
}

func Test_GenerateCloudfrontDistributions(t *testing.T) {
	cases := []struct {
		name    string
		results []core.CloudResource
		cfg     config.Application
		data    TemplateData
		want    []*resources.CloudfrontDistribution
	}{
		{
			name: "simple gateway test",
			results: []core.CloudResource{
				&core.Gateway{
					Name:   "gw",
					GWType: core.GatewayKind,
					Routes: []core.Route{{Path: "/"}},
				},
			},
			cfg: config.Application{
				Provider: "aws",
				AppName:  "app",
				Exposed: map[string]*config.Expose{
					"gw": {
						Type:                   "apigateway",
						ContentDeliveryNetwork: config.ContentDeliveryNetwork{Id: "distro"},
					},
				},
			},
			data: TemplateData{
				TemplateData: provider.TemplateData{
					Gateways: []provider.Gateway{
						{Name: "gw"},
					},
				},
			},
			want: []*resources.CloudfrontDistribution{
				{
					Id: "app-distro",
					Origins: []core.ResourceKey{
						{Kind: core.GatewayKind, Name: "gw"},
					},
				},
			},
		},
		{
			name: "simple static unit test",
			results: []core.CloudResource{
				&core.StaticUnit{
					Name: "su",
				},
			},
			cfg: config.Application{
				Provider: "aws",
				AppName:  "app",
				StaticUnit: map[string]*config.StaticUnit{
					"su": {
						Type:                   "apigateway",
						ContentDeliveryNetwork: config.ContentDeliveryNetwork{Id: "distro"},
					},
				},
			},
			data: TemplateData{
				TemplateData: provider.TemplateData{
					StaticUnits: []provider.StaticUnit{
						{Name: "su"},
					},
				},
			},
			want: []*resources.CloudfrontDistribution{
				{
					Id: "app-distro",
					Origins: []core.ResourceKey{
						{Kind: core.StaticUnitKind, Name: "su"},
					},
				},
			},
		},
		{
			name: "simple static unit with index document test",
			results: []core.CloudResource{
				&core.StaticUnit{
					Name:          "su",
					IndexDocument: "index.html",
				},
			},
			cfg: config.Application{
				Provider: "aws",
				AppName:  "app",
				StaticUnit: map[string]*config.StaticUnit{
					"su": {
						Type:                   "apigateway",
						ContentDeliveryNetwork: config.ContentDeliveryNetwork{Id: "distro"},
					},
				},
			},
			data: TemplateData{
				TemplateData: provider.TemplateData{
					StaticUnits: []provider.StaticUnit{
						{Name: "su"},
					},
				},
			},
			want: []*resources.CloudfrontDistribution{
				{
					Id: "app-distro",
					Origins: []core.ResourceKey{
						{Kind: core.StaticUnitKind, Name: "su"},
					},
					DefaultRootObject: "index.html",
				},
			},
		},
		{
			name: "static unit and gw test",
			results: []core.CloudResource{
				&core.StaticUnit{
					Name: "su",
				},
				&core.Gateway{
					Name:   "gw",
					GWType: core.GatewayKind,
					Routes: []core.Route{{Path: "/"}},
				},
			},
			cfg: config.Application{
				Provider: "aws",
				AppName:  "app",
				StaticUnit: map[string]*config.StaticUnit{
					"su": {
						Type:                   "apigateway",
						ContentDeliveryNetwork: config.ContentDeliveryNetwork{Id: "distro"},
					},
				},
				Exposed: map[string]*config.Expose{
					"gw": {
						Type:                   "apigateway",
						ContentDeliveryNetwork: config.ContentDeliveryNetwork{Id: "distro"},
					},
				},
			},
			data: TemplateData{
				TemplateData: provider.TemplateData{
					StaticUnits: []provider.StaticUnit{
						{Name: "su"},
					},
					Gateways: []provider.Gateway{
						{Name: "gw"},
					},
				},
			},
			want: []*resources.CloudfrontDistribution{
				{
					Id: "app-distro",
					Origins: []core.ResourceKey{
						{Kind: core.GatewayKind, Name: "gw"},
						{Kind: core.StaticUnitKind, Name: "su"},
					},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result := core.CompilationResult{}

			result.AddAll(tt.results)

			aws := AWS{
				Config: &tt.cfg,
			}
			aws.GenerateCloudfrontDistributions(&tt.data, &result)
			for _, cf := range tt.want {
				found := false
				for _, gotCf := range tt.data.CloudfrontDistributions {
					if gotCf.Id == cf.Id {
						found = true
						assert.Equal(cf.DefaultRootObject, gotCf.DefaultRootObject)
						for _, cfOrigin := range cf.Origins {
							originFound := false
							for _, gotCfOrigin := range gotCf.Origins {
								if cfOrigin.String() == gotCfOrigin.String() {
									originFound = true
								}
							}
							assert.True(originFound)
						}
					}
				}
				assert.True(found)
			}
		})
	}
}
