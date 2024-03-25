import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as docker from '@pulumi/docker'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Image: docker.Image
    ExecutionRole: aws.iam.Role
    EnvironmentVariables: ModelCaseWrapper<Record<string, pulumi.Output<string>>>
    Subnets: aws.ec2.Subnet[]
    SecurityGroups: aws.ec2.SecurityGroup[]
    MemorySize: pulumi.Input<number>
    Timeout: pulumi.Input<number>
    EfsAccessPoint: aws.efs.AccessPoint
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lambda.Function {
    return new aws.lambda.Function(
        args.Name,
        {
            packageType: 'Image',
            imageUri: args.Image.imageName,
            //TMPL {{- if .MemorySize }}
            memorySize: args.MemorySize,
            //TMPL {{- end }}
            //TMPL {{- if .Timeout }}
            timeout: args.Timeout,
            //TMPL {{- end }}
            role: args.ExecutionRole.arn,
            name: args.Name,
            //TMPL {{- if .EfsAccessPoint }}
            fileSystemConfig: {
                arn: args.EfsAccessPoint.arn,
                localMountPath: args.EfsAccessPoint.rootDirectory.path,
            },
            //TMPL {{- end }}
            //TMPL {{- if and .SecurityGroups .Subnets }}
            vpcConfig: {
                securityGroupIds: args.SecurityGroups.map((sg) => sg.id),
                subnetIds: args.Subnets.map((subnet) => subnet.id),
            },
            //TMPL {{- end }}
            //TMPL {{- if .EnvironmentVariables }}
            environment: {
                variables: args.EnvironmentVariables,
            },
            //TMPL {{- end }}
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
        {
            dependsOn: args.dependsOn,
        }
    )
}

function properties(object: aws.lambda.Function, args: Args) {
    return {
        LambdaIntegrationUri: object.invokeArn,
        Arn: object.arn,
    }
}

function importResource(args: Args): aws.lambda.Function {
    return aws.lambda.Function.get(args.Name, args.Id)
}
