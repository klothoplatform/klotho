import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    // lambdaName: string,
    // image: pulumi.Output<string>,
    Role: aws.iam.Role
    EnvironmentVariables: Record<string, pulumi.Output<string>>
    // dependsOn: []
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lambda.Function {
    return new aws.lambda.Function(
        args.Name,
        {
            packageType: 'Image',
            imageUri: 'TODO-image-uri',
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
