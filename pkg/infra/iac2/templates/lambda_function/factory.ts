import * as aws from '@pulumi/aws'
import { LogGroup } from '@pulumi/aws/cloudwatch'

interface Args {
    Name: string
    // lambdaName: string,
    // image: pulumi.Output<string>,
    CloudwatchGroup: LogGroup
    Role: aws.iam.Role
    // envVars: Record<string, pulumi.Output<string>>,
    // dependsOn: []
    dependsOn: any[]
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
