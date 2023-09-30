import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as docker from '@pulumi/docker'

interface Args {
    Name: string
    Image: docker.Image
    ExecutionRole: aws.iam.Role
    EnvironmentVariables: Record<string, pulumi.Output<string>>
    Subnets: aws.ec2.Subnet[]
    SecurityGroups: aws.ec2.SecurityGroup[]
    MemorySize: pulumi.Input<number>
    Timeout: pulumi.Input<number>
    EfsAccessPoint: aws.efs.AccessPoint
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
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
                localMountPath: `/mnt/${args.EfsAccessPoint.rootDirectory.path}`,
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
            tags: {
                env: 'production',
                service: args.Name,
            },
        },
        {
            dependsOn: args.dependsOn,
        }
    )
}
