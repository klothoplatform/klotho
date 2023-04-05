import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as docker from '@pulumi/docker'

interface Args {
    Name: string
    Image: docker.Image
    Role: aws.iam.Role
    EnvironmentVariables: Record<string, pulumi.Output<string>>
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lambda.Function {
    return new aws.lambda.Function(
        args.Name,
        {
            packageType: 'Image',
            imageUri: args.Image.imageName,
            role: args.Role.arn,
            name: args.Name,
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
