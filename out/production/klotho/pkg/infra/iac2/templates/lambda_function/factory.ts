import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as docker from '@pulumi/docker'

interface Args {
    Name: string
    Image: docker.Image
    Role: aws.iam.Role
    EnvironmentVariables: Record<string, pulumi.Output<string>>
    Subnets: aws.ec2.Subnet[]
    SecurityGroups: aws.ec2.SecurityGroup[]
    MemorySize: pulumi.Input<number>
    Timeout: pulumi.Input<number>
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lambda.Function {
    return new aws.lambda.Function(
        args.Name,
        {
            packageType: 'Image',
            imageUri: args.Image.imageName,
            //TMPL {{- if .MemorySize.Raw }}
            memorySize: args.MemorySize,
            //TMPL {{- end }}
            //TMPL {{- if .Timeout.Raw }}
            timeout: args.Timeout,
            //TMPL {{- end }}
            role: args.Role.arn,
            name: args.Name,
            //TMPL {{- if and .SecurityGroups.Raw .Subnets.Raw }}
            vpcConfig: {
                securityGroupIds: args.SecurityGroups.map((sg) => sg.id),
                subnetIds: args.Subnets.map((subnet) => subnet.id),
            },
            //TMPL {{- end }}
            environment: {
                variables: args.EnvironmentVariables,
            },
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
