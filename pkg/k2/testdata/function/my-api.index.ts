import * as aws from '@pulumi/aws'
import * as path from 'path'
import * as pulumi from '@pulumi/pulumi'


const kloConfig = new pulumi.Config('klo')
const protect = kloConfig.getBoolean('protect') ?? false
const awsConfig = new pulumi.Config('aws')
const awsProfile = awsConfig.get('profile')
const accountId = pulumi.output(aws.getCallerIdentity({}))
const region = pulumi.output(aws.getRegion({}))

const docker_func_function = aws.lambda.Function.get("docker-func-function", "preview(id=aws:lambda_function:docker-func-function)")
const my_api_api = new aws.apigateway.RestApi("my-api-api", {
        binaryMediaTypes: ["application/octet-stream", "image/*"],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-api-api"},
    })
const docker_func_api_method = new aws.apigateway.Method(
        "docker-func-api_method",
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
const docker_func_docker_func_function = new aws.lambda.Permission("docker-func-docker-func-function", {
        action: "lambda:InvokeFunction",
        function: docker_func_function.name,
        principal: "apigateway.amazonaws.com",
        sourceArn: pulumi.interpolate`${my_api_api.executionArn}/*`,
    })
const docker_func = new aws.apigateway.Integration(
        "docker-func",
        {
            restApi: my_api_api.id,
            resourceId: my_api_api.rootResourceId,
            httpMethod: docker_func_api_method.httpMethod,
            integrationHttpMethod: "POST",
            type: "AWS_PROXY",
            uri: docker_func_function.invokeArn,
        },
        { parent: docker_func_api_method }
    )
const api_deployment_0 = new aws.apigateway.Deployment(
        "api_deployment-0",
        {
            restApi: my_api_api.id,
            triggers: {dockerFunc: "docker-func", dockerFuncApiMethod: "docker-func-api_method"},
        },
        {
            dependsOn: [docker_func, docker_func_api_method, my_api_api],
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
	"aws:lambda_function:docker-func-function": (docker_func_function as any).urn,
	"aws:rest_api:my-api-api": (my_api_api as any).urn,
	"aws:api_method:my-api-api:docker-func-api_method": (docker_func_api_method as any).urn,
	"aws:lambda_permission:docker-func-docker-func-function": (docker_func_docker_func_function as any).urn,
	"aws:api_integration:my-api-api:docker-func": (docker_func as any).urn,
	"aws:api_deployment:my-api-api:api_deployment-0": (api_deployment_0 as any).urn,
	"aws:api_stage:my-api-api:my-api-stage": (my_api_stage as any).urn,
}
