import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    RestApi: aws.apigateway.RestApi
    Deployment: aws.apigateway.Deployment
    StageName: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.Stage {
    return new aws.apigateway.Stage(args.Name, {
        deployment: args.Deployment.id,
        restApi: args.RestApi.id,
        stageName: args.StageName,
    })
}

function properties(object: aws.apigateway.Stage, args: Args) {
    return {
        StageInvokeUrl: object.invokeUrl.apply((d) => d.split('//')[1].split('/')[0]),
    }
}

function infraExports(
    object: ReturnType<typeof create>,
    args: Args,
    props: ReturnType<typeof properties>
) {
    return {
        InvokeUrl: props.StageInvokeUrl,
    }
}
