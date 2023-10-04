import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as awsInputs from '@pulumi/aws/types/input'

interface Args {
    Name: string
    Attributes: Record<string, string>
    HashKey: string
    RangeKey: string
    BillingMode: string
    protect: boolean
}

function create(args: Args): aws.dynamodb.Table {
    return new aws.dynamodb.Table(
        args.Name,
        {
            attributes: Object.entries(args.Attributes).map((attribute) => {
                return {
                    name: attribute['Name'],
                    type: attribute['Type'],
                }
            }),
            hashKey: args.HashKey,
            //TMPL {{- if .RangeKey }}
            rangeKey: args.RangeKey,
            //TMPL {{- end }}
            billingMode: args.BillingMode,
        },
        { protect: args.protect }
    )
}

function properties(object: aws.dynamodb.Table, args: Args) {
    return {
        DynamoTableStreamArn: pulumi.interpolate`${object.arn}/stream/*`,
        DynamoTableBackupArn: pulumi.interpolate`${object.arn}/backup/*`,
        DynamoTableExportArn: pulumi.interpolate`${object.arn}/export/*`,
        DynamoTableIndexArn: pulumi.interpolate`${object.arn}/index/*`,
    }
}
