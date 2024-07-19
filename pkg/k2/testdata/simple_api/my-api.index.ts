import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import * as pulumi from '@pulumi/pulumi'
import { OutputInstance } from '@pulumi/pulumi'


const kloConfig = new pulumi.Config('klo')
const protect = kloConfig.getBoolean('protect') ?? false
const awsConfig = new pulumi.Config('aws')
const awsProfile = awsConfig.get('profile')
const accountId = pulumi.output(aws.getCallerIdentity({}))
const region = pulumi.output(aws.getRegion({}))

const ecs_cluster_0 = aws.ecs.Cluster.get("ecs_cluster-0", "preview(id=aws:ecs_cluster:ecs_cluster-0)")
const my_api_api = new aws.apigateway.RestApi("my-api-api", {
        binaryMediaTypes: ["application/octet-stream", "image/*"],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-api-api"},
    })
const default_network_vpc = aws.ec2.Vpc.get("default-network-vpc", "preview(id=aws:vpc:default-network-vpc)")
const __any_method = new aws.apigateway.Method(
        "--any-method",
        {
            restApi: my_api_api.id,
            resourceId: my_api_api.rootResourceId,
            httpMethod: "ANY",
            authorization: "NONE",
        },
        {
            parent: my_api_api
        }
    )
const my_container_service_security_group = aws.ec2.SecurityGroup.get("my-container-service-security_group", "preview(id=aws:security_group:default-network-vpc:my-container-service-security_group)")
const default_network_private_subnet_1 = aws.ec2.Subnet.get("default-network-private-subnet-1", "preview(id=aws:subnet:default-network-vpc:default-network-private-subnet-1)")
const default_network_private_subnet_2 = aws.ec2.Subnet.get("default-network-private-subnet-2", "preview(id=aws:subnet:default-network-vpc:default-network-private-subnet-2)")
const default_network_public_subnet_1 = aws.ec2.Subnet.get("default-network-public-subnet-1", "preview(id=aws:subnet:default-network-vpc:default-network-public-subnet-1)")
const default_network_public_subnet_2 = aws.ec2.Subnet.get("default-network-public-subnet-2", "preview(id=aws:subnet:default-network-vpc:default-network-public-subnet-2)")
const my_container_tg = aws.lb.TargetGroup.get("my-container-tg", "preview(id=aws:target_group:my-container-tg)")
const api_my_container_lb = aws.lb.LoadBalancer.get("api-my-container-lb", "preview(id=aws:load_balancer:api-my-container-lb)")
export const api_my_container_lb_DomainName = api_my_container_lb.dnsName
const my_container_service = aws.ecs.Service.get("my-container-service", "preview(id=aws:ecs_service:my-container-service)".split('/').slice(-2).join('/'))
const __any_integration_api_my_container_lb = new aws.apigateway.VpcLink("--any-integration-api-my-container-lb", {
        targetArn: api_my_container_lb.arn,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "--any-integration-api-my-container-lb"},
    })
const __any_integration = new aws.apigateway.Integration(
        "--any-integration",
        {
            restApi: my_api_api.id,
            resourceId: my_api_api.rootResourceId,
            httpMethod: __any_method.httpMethod,
            integrationHttpMethod: "ANY",
            type: "HTTP_PROXY",
            connectionType: "VPC_LINK",
            connectionId: __any_integration_api_my_container_lb.id,
            uri: pulumi.interpolate`http://${
            (api_my_container_lb as aws.lb.LoadBalancer).dnsName
        }${"/".replace('+', '')}`,
        },
        { parent: __any_method }
    )
const api_deployment_0 = new aws.apigateway.Deployment(
        "api_deployment-0",
        {
            restApi: my_api_api.id,
            triggers: {AnyIntegration: "--any-integration", AnyMethod: "--any-method"},
        },
        {
            dependsOn: [__any_integration, __any_method, my_api_api],
        }
    )
const my_api_stage = new aws.apigateway.Stage("my-api-stage", {
        deployment: api_deployment_0.id,
        restApi: my_api_api.id,
        stageName: "api",
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-api-stage"},
    })
export const my_api_stage_Url = my_api_stage.invokeUrl

export const $outputs = {
	Endpoint: my_api_stage.invokeUrl,
}

export const $urns = {
	"aws:ecs_cluster:ecs_cluster-0": (ecs_cluster_0 as any).urn,
	"aws:rest_api:my-api-api": (my_api_api as any).urn,
	"aws:vpc:default-network-vpc": (default_network_vpc as any).urn,
	"aws:api_method:my-api-api:--any-method": (__any_method as any).urn,
	"aws:security_group:default-network-vpc:my-container-service-security_group": (my_container_service_security_group as any).urn,
	"aws:subnet:default-network-vpc:default-network-private-subnet-1": (default_network_private_subnet_1 as any).urn,
	"aws:subnet:default-network-vpc:default-network-private-subnet-2": (default_network_private_subnet_2 as any).urn,
	"aws:subnet:default-network-vpc:default-network-public-subnet-1": (default_network_public_subnet_1 as any).urn,
	"aws:subnet:default-network-vpc:default-network-public-subnet-2": (default_network_public_subnet_2 as any).urn,
	"aws:target_group:my-container-tg": (my_container_tg as any).urn,
	"aws:load_balancer:api-my-container-lb": (api_my_container_lb as any).urn,
	"aws:ecs_service:my-container-service": (my_container_service as any).urn,
	"aws:vpc_link:--any-integration-api-my-container-lb": (__any_integration_api_my_container_lb as any).urn,
	"aws:api_integration:my-api-api:--any-integration": (__any_integration as any).urn,
	"aws:api_deployment:my-api-api:api_deployment-0": (api_deployment_0 as any).urn,
	"aws:api_stage:my-api-api:my-api-stage": (my_api_stage as any).urn,
}
