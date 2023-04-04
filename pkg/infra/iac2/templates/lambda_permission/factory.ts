import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
interface Args {
    Name: string
    Function: aws.lambda.Function
    Principal: string
    Source: pulumi.Output<string>
    Action: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lambda.Permission {
    return new aws.lambda.Permission(args.Name, {
        action: args.Action,
        function: args.Function.name,
        principal: args.Principal,
        sourceArn: args.Source,
    })
}
