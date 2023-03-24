import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { LogGroup } from '@pulumi/aws/cloudwatch'
import { Role } from '@pulumi/aws/iam'

interface Args {
    ExecUnitName: string
    // lambdaName: string,
    // image: pulumi.Output<string>,
    CloudwatchGroup: LogGroup
    // lambdaRole: Role,
    // envVars: Record<string, pulumi.Output<string>>,
    // dependsOn: []
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lambda.Function {
    return new aws.lambda.Function(
        args.ExecUnitName,
        {
            packageType: 'Image',
            imageUri: 'TODO-image-uri', //args.image,
            role: 'TODO-role', //args.lambdaRole.arn,
            name: 'TODO-lambda-name', //args.lambdaName,
            tags: {
                env: 'production',
                service: args.ExecUnitName,
            },
        },
        {
            dependsOn: [args.CloudwatchGroup],
        }
    )
}
