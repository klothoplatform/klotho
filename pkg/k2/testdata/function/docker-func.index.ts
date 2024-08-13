import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import * as docker from '@pulumi/docker'
import * as path from 'path'
import * as pulumi from '@pulumi/pulumi'


const kloConfig = new pulumi.Config('klo')
const protect = kloConfig.getBoolean('protect') ?? false
const awsConfig = new pulumi.Config('aws')
const awsProfile = awsConfig.get('profile')
const accountId = pulumi.output(aws.getCallerIdentity({}))
const region = pulumi.output(aws.getRegion({}))

const docker_func_image_ecr_repo = new aws.ecr.Repository("docker-func-image-ecr_repo", {
        imageScanningConfiguration: {
            scanOnPush: true,
        },
        imageTagMutability: 'MUTABLE',
        forceDelete: true,
        encryptionConfigurations: [{ encryptionType: 'KMS' }],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "docker-func-image-ecr_repo"},
    })
const docker_func_function_executionrole = new aws.iam.Role("docker-func-function-ExecutionRole", {
        assumeRolePolicy: pulumi.jsonStringify({Statement: [{Action: ["sts:AssumeRole"], Effect: "Allow", Principal: {Service: ["lambda.amazonaws.com"]}}], Version: "2012-10-17"}),
        managedPolicyArns: [
            ...["arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"],
        ],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "docker-func-function-ExecutionRole"},
    })
const docker_func_image = (() => {
        const base = new docker.Image(`${"docker-func-image"}-base`, {
            build: {
                context: "/",
                dockerfile: "/Dockerfile",
                platform: "linux/amd64",
            },
            skipPush: true,
            imageName: pulumi.interpolate`${docker_func_image_ecr_repo.repositoryUrl}:base`,
        })

        const sha256 = base.repoDigest.apply((digest) => {
            return digest.substring(digest.indexOf('sha256:') + 7)
        })

        return new docker.Image(
            "docker-func-image",
            {
                build: {
                    context: "/",
                    dockerfile: "/Dockerfile",
                    platform: "linux/amd64",
                    cacheFrom: {
                        images: [base.imageName],
                    },
                },
                registry: aws.ecr
                    .getAuthorizationTokenOutput(
                        { registryId: docker_func_image_ecr_repo.registryId },
                        { async: true }
                    )
                    .apply((registryToken) => {
                        return {
                            server: docker_func_image_ecr_repo.repositoryUrl,
                            username: registryToken.userName,
                            password: registryToken.password,
                        }
                    }),
                imageName: pulumi.interpolate`${docker_func_image_ecr_repo.repositoryUrl}:${sha256}`,
            },
            { parent: base }
        )
    })()
const docker_func_function = new aws.lambda.Function(
        "docker-func-function",
        {
            packageType: 'Image',
            imageUri: docker_func_image.imageName,
            memorySize: 128,
            timeout: 3,
            role: docker_func_function_executionrole.arn,
            tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "docker-func-function"},
            loggingConfig: {
                logFormat: "Text",
            },
        },
        {
            dependsOn: [docker_func_function_executionrole, docker_func_image],
        }
    )
const docker_func_function_log_group = new aws.cloudwatch.LogGroup("docker-func-function-log_group", {
        name: pulumi.interpolate`/aws/lambda/${docker_func_function.name}`,
        retentionInDays: 5,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "docker-func-function-log_group"},
    })

export const $outputs = {
	FunctionArn: docker_func_function.arn,
	FunctionName: docker_func_function.name,
}

export const $urns = {
	"aws:ecr_repo:docker-func-image-ecr_repo": (docker_func_image_ecr_repo as any).urn,
	"aws:iam_role:docker-func-function-ExecutionRole": (docker_func_function_executionrole as any).urn,
	"aws:ecr_image:docker-func-image": (docker_func_image as any).urn,
	"aws:lambda_function:docker-func-function": (docker_func_function as any).urn,
	"aws:log_group:docker-func-function-log_group": (docker_func_function_log_group as any).urn,
}
