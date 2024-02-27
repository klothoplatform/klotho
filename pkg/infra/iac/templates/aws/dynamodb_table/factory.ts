import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as awsInputs from '@pulumi/aws/types/input'
import { TemplateWrapper } from '../../wrappers'

interface Args {
    Name: string
    Attributes: TemplateWrapper<pulumi.Input<pulumi.Input<awsInputs.dynamodb.TableAttribute>[]>>
    HashKey: string
    RangeKey: string
    BillingMode: string
    protect: boolean
    Tags: ModelCaseWrapper<Record<string, string>>
}

function create(args: Args): aws.dynamodb.Table {
    return new aws.dynamodb.Table(
        args.Name,
        {
            attributes: args.Attributes,
            hashKey: args.HashKey,
            //TMPL {{- if .RangeKey }}
            rangeKey: args.RangeKey,
            //TMPL {{- end }}
            billingMode: args.BillingMode,
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
        { protect: args.protect }
    )
}

function properties(object: aws.dynamodb.Table, args: Args) {
    return {
        Arn: object.arn,
        DynamoTableStreamArn: pulumi.interpolate`${object.arn}/stream/*`,
        DynamoTableBackupArn: pulumi.interpolate`${object.arn}/backup/*`,
        DynamoTableExportArn: pulumi.interpolate`${object.arn}/export/*`,
        DynamoTableIndexArn: pulumi.interpolate`${object.arn}/index/*`,
        Name: object.name,
    }
}
