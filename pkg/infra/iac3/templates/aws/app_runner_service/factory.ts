import * as aws from '@pulumi/aws'
import * as docker from '@pulumi/docker'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    Image: docker.Image
    InstanceRole: aws.iam.Role
    EnvironmentVariables: Record<string, pulumi.Output<string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apprunner.Service {
    return new aws.apprunner.Service(args.Name, {
        serviceName: args.Name,
        sourceConfiguration: {
            authenticationConfiguration: {
                accessRoleArn: args.InstanceRole.arn,
            },
            imageRepository: {
                imageIdentifier: args.Image.imageName,
                imageRepositoryType: 'ECR',
                imageConfiguration: {
                    runtimeEnvironmentVariables: args.EnvironmentVariables,
                },
            },
        },
        instanceConfiguration: {
            instanceRoleArn: args.InstanceRole.arn,
        },
        networkConfiguration: {
            egressConfiguration: {
                egressType: 'DEFAULT',
            },
            ingressConfiguration: {
                isPubliclyAccessible: true,
            },
        },
    })
}
