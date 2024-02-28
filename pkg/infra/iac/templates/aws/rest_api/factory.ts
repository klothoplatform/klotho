import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    BinaryMediaTypes: string[]
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.RestApi {
    return new aws.apigateway.RestApi(args.Name, {
        binaryMediaTypes: args.BinaryMediaTypes,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.apigateway.RestApi, args: Args) {
    return {
        ChildResources: pulumi.interpolate`${object.executionArn}/*`,
    }
}
