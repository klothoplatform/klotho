import * as aws from '@pulumi/aws'
import * as docker from '@pulumi/docker'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Image: docker.Image
    InstanceRole: aws.iam.Role
    EnvironmentVariables: ModelCaseWrapper<Record<string, pulumi.Output<string>>>
    Port: number
    Tags: ModelCaseWrapper<Record<string, string>>
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
                //TMPL {{- if (or .Port .EnvironmentVariables) }}
                imageConfiguration: {
                    //TMPL {{- if .Port }}
                    port: 'args.Port',
                    //TMPL {{- end }}
                    //TMPL {{- if .EnvironmentVariables }}
                    runtimeEnvironmentVariables: args.EnvironmentVariables,
                    //TMPL {{- end }}
                },
                //TMPL {{- end }}
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
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: ReturnType<typeof create>, args: Args) {
    return {}
}

function infraExports(
    object: ReturnType<typeof create>,
    args: Args,
    props: ReturnType<typeof properties>
) {
    return {
        Url: object.serviceUrl,
    }
}
