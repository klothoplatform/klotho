package aws

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_ExpandExecutionUnit(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test", DockerfilePath: "path"}
	cases := []struct {
		name          string
		unit          *core.ExecutionUnit
		chart         *kubernetes.HelmChart
		constructType string
		config        *config.Application
		want          coretesting.ResourcesExpectation
	}{
		{
			name:          "single lambda exec unit",
			unit:          eu,
			constructType: "lambda_function",
			config:        &config.Application{AppName: "my-app", Defaults: config.Defaults{ExecutionUnit: config.KindDefaults{Type: Lambda}}},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_image:my-app-test",
					"aws:ecr_repo:my-app",
					"aws:iam_role:my-app-test-ExecutionRole",
					"aws:lambda_function:my-app-test",
					"aws:log_group:my-app-test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecr_image:my-app-test", Destination: "aws:ecr_repo:my-app"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:ecr_image:my-app-test"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:iam_role:my-app-test-ExecutionRole"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:log_group:my-app-test"},
				},
			},
		},
		{
			name: "single k8s exec unit",
			unit: eu,
			chart: &kubernetes.HelmChart{
				ExecutionUnits: []*kubernetes.HelmExecUnit{{Name: eu.Name}},
				ProviderValues: []kubernetes.HelmChartValue{
					{
						ExecUnitName: eu.Name,
						Type:         string(kubernetes.ServiceAccountAnnotationTransformation),
						Key:          "ROLE",
					},
					{
						ExecUnitName: eu.Name,
						Type:         string(kubernetes.ImageTransformation),
						Key:          "IMAGE",
					},
					{
						ExecUnitName: eu.Name,
						Type:         string(kubernetes.InstanceTypeValue),
						Key:          "InstanceType",
					},
				},
				Values: make(map[string]any),
			},
			constructType: kubernetes.HELM_CHART_TYPE,
			config: &config.Application{AppName: "my-app",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {
						Type:             kubernetes.KubernetesType,
						NetworkPlacement: "private",
						InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{
							InstanceType: "t3.medium",
							DiskSizeGiB:  20,
							ClusterId:    "cluster1",
						}),
					},
				},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:ecr_image:my-app-test",
					"aws:ecr_repo:my-app",
					"aws:eks_addon:my-app-cluster1-addon-vpc-cni",
					"aws:eks_cluster:my-app-cluster1",
					"aws:eks_node_group:my-app_cluster1_private_t3_medium",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-cluster1-ClusterAdmin",
					"aws:iam_role:my-app-cluster1_private_t3_medium-NodeRole",
					"aws:iam_role:my-app-my-app-test-ExecutionRole",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
					"kubernetes:helm_chart:",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecr_image:my-app-test", Destination: "aws:ecr_repo:my-app"},
					{Source: "aws:eks_addon:my-app-cluster1-addon-vpc-cni", Destination: "aws:eks_cluster:my-app-cluster1"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:iam_role:my-app-cluster1-ClusterAdmin"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:vpc:my_app"},
					{Source: "aws:eks_node_group:my-app_cluster1_private_t3_medium", Destination: "aws:eks_cluster:my-app-cluster1"},
					{Source: "aws:eks_node_group:my-app_cluster1_private_t3_medium", Destination: "aws:iam_role:my-app-cluster1_private_t3_medium-NodeRole"},
					{Source: "aws:eks_node_group:my-app_cluster1_private_t3_medium", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_node_group:my-app_cluster1_private_t3_medium", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
					{Source: "kubernetes:helm_chart:", Destination: "aws:ecr_image:my-app-test"},
					{Source: "kubernetes:helm_chart:", Destination: "aws:eks_cluster:my-app-cluster1"},
					{Source: "kubernetes:helm_chart:", Destination: "aws:eks_node_group:my-app_cluster1_private_t3_medium"},
					{Source: "kubernetes:helm_chart:", Destination: "aws:iam_role:my-app-my-app-test-ExecutionRole"},
				},
			},
		},
		{
			name: "single fargate k8s exec unit",
			unit: eu,
			chart: &kubernetes.HelmChart{
				ExecutionUnits: []*kubernetes.HelmExecUnit{{Name: eu.Name}},
				Values:         make(map[string]any),
			},
			constructType: kubernetes.HELM_CHART_TYPE,
			config: &config.Application{AppName: "my-app",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {
						Type:             kubernetes.KubernetesType,
						NetworkPlacement: "private",
						InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{
							NodeType:     "fargate",
							InstanceType: "t3.medium",
							DiskSizeGiB:  20,
							ClusterId:    "cluster1",
						}),
					},
				},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_addon:my-app-cluster1-addon-vpc-cni",
					"aws:eks_cluster:my-app-cluster1",
					"aws:eks_fargate_profile:my-app_klotho-fargate-profile_private",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-cluster1-ClusterAdmin",
					"aws:iam_role:my-app-klotho-fargate-profile-PodExecutionRole",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
					"kubernetes:helm_chart:",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:my-app-cluster1-addon-vpc-cni", Destination: "aws:eks_cluster:my-app-cluster1"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:iam_role:my-app-cluster1-ClusterAdmin"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:eks_cluster:my-app-cluster1", Destination: "aws:vpc:my_app"},
					{Source: "aws:eks_fargate_profile:my-app_klotho-fargate-profile_private", Destination: "aws:eks_cluster:my-app-cluster1"},
					{Source: "aws:eks_fargate_profile:my-app_klotho-fargate-profile_private", Destination: "aws:iam_role:my-app-klotho-fargate-profile-PodExecutionRole"},
					{Source: "aws:eks_fargate_profile:my-app_klotho-fargate-profile_private", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_fargate_profile:my-app_klotho-fargate-profile_private", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
					{Source: "kubernetes:helm_chart:", Destination: "aws:eks_cluster:my-app-cluster1"},
					{Source: "kubernetes:helm_chart:", Destination: "aws:eks_fargate_profile:my-app_klotho-fargate-profile_private"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.chart != nil {
				dag.AddResource(tt.chart)
			}

			aws := AWS{
				Config: tt.config,
			}
			err := aws.expandExecutionUnit(dag, tt.unit, tt.constructType)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
			res, _ := aws.GetResourcesDirectlyTiedToConstruct(tt.unit)
			assert.NotEmpty(res)
		})
	}
}

func Test_handleHelmChartAwsValues(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test", DockerfilePath: "path"}
	config := &config.Application{AppName: "my-app",
		ExecutionUnits: map[string]*config.ExecutionUnit{
			"test": {
				Type:             kubernetes.KubernetesType,
				NetworkPlacement: "private",
				InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{
					InstanceType: "t3.medium",
					DiskSizeGiB:  20,
					ClusterId:    "cluster1",
				}),
			},
		},
	}
	type testResult struct {
		params map[string]any
		values map[string]any
	}
	cases := []struct {
		name  string
		unit  *core.ExecutionUnit
		value kubernetes.HelmChartValue
		want  testResult
	}{
		{
			name: "ImageTransformation",
			unit: eu,
			value: kubernetes.HelmChartValue{
				ExecUnitName: eu.Name,
				Type:         string(kubernetes.ImageTransformation),
				Key:          "IMAGE",
			},
			want: testResult{
				params: map[string]any{
					"IMAGE": resources.ImageCreateParams{
						AppName: config.AppName,
						Refs:    core.BaseConstructSetOf(eu),
						Name:    eu.Name,
					},
				},
				values: map[string]any{
					"IMAGE": core.IaCValue{
						Resource: &resources.EcrImage{},
						Property: resources.ECR_IMAGE_NAME_IAC_VALUE,
					},
				},
			},
		},
		{
			name: "ServiceAccountAnnotationTransformation",
			unit: eu,
			value: kubernetes.HelmChartValue{
				ExecUnitName: eu.Name,
				Type:         string(kubernetes.ServiceAccountAnnotationTransformation),
				Key:          "SERVICEACCOUNT",
			},
			want: testResult{
				params: map[string]any{
					"SERVICEACCOUNT": resources.RoleCreateParams{
						Name:    fmt.Sprintf("%s-%s-ExecutionRole", config.AppName, eu.Name),
						Refs:    core.BaseConstructSetOf(eu),
						AppName: config.AppName,
					},
				},
				values: map[string]any{
					"SERVICEACCOUNT": core.IaCValue{
						Resource: &resources.IamRole{},
						Property: resources.ARN_IAC_VALUE,
					},
				},
			},
		},
		{
			name: "InstanceTypeKey",
			unit: eu,
			value: kubernetes.HelmChartValue{
				ExecUnitName: eu.Name,
				Type:         string(kubernetes.InstanceTypeKey),
				Key:          "SERVICEACCOUNT",
			},
			want: testResult{
				params: make(map[string]any),
				values: map[string]any{
					"SERVICEACCOUNT": core.IaCValue{
						Property: "eks.amazonaws.com/nodegroup",
					},
				},
			},
		},
		{
			name: "InstanceTypeValue",
			unit: eu,
			value: kubernetes.HelmChartValue{
				ExecUnitName: eu.Name,
				Type:         string(kubernetes.InstanceTypeValue),
				Key:          "InstanceTypeValue",
			},
			want: testResult{
				params: map[string]any{
					"InstanceTypeValue": resources.EksNodeGroupCreateParams{
						NetworkType:  "private",
						InstanceType: "t3.medium",
						ClusterName:  "cluster1",
						AppName:      config.AppName,
						Refs:         core.BaseConstructSetOf(eu),
					},
				},
				values: map[string]any{
					"InstanceTypeValue": core.IaCValue{
						Resource: &resources.EksNodeGroup{},
						Property: resources.NODE_GROUP_NAME_IAC_VALUE,
					},
				},
			},
		},
		{
			name: "TargetGroupTransformation",
			unit: eu,
			value: kubernetes.HelmChartValue{
				ExecUnitName: eu.Name,
				Type:         string(kubernetes.TargetGroupTransformation),
				Key:          "TargetGroupTransformation",
			},
			want: testResult{
				params: map[string]any{
					"TargetGroupTransformation": resources.TargetGroupCreateParams{
						AppName: config.AppName,
						Refs:    core.BaseConstructSetOf(eu),
						Name:    eu.Name,
					},
				},
				values: map[string]any{
					"TargetGroupTransformation": core.IaCValue{
						Resource: &resources.TargetGroup{},
						Property: resources.ARN_IAC_VALUE,
					},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			chart := &kubernetes.HelmChart{}
			chart.Values = make(map[string]any)
			chart.ProviderValues = append(chart.ProviderValues, tt.value)

			aws := AWS{
				Config: config,
			}
			result, err := aws.handleHelmChartAwsValues(chart, tt.unit, core.NewResourceGraph())
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.values, chart.Values)
			if tt.value.Type == string(kubernetes.ServiceAccountAnnotationTransformation) {
				assert.Equal(tt.want.params["RoleName"], result["RoleName"])
				assert.Equal(tt.want.params["Refs"], result["Refs"])
			} else {
				assert.Equal(tt.want.params, result)
			}
		})
	}
}
