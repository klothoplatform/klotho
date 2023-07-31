import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as awsInputs from '@pulumi/aws/types/input'

interface Args {
    Name: string
    Attributes: pulumi.Input<pulumi.Input<inputs.dynamodb.TableAttribute>[]>
    HashKey: string
    RangeKey: string
    BillingMode: string
    protect: boolean
}

function create(args: Args): aws.dynamodb.Table {
    return new aws.dynamodb.Table(
        args.Name,
        {
            attributes: args.Attributes,
            hashKey: args.HashKey,
            //TMPL {{- if .RangeKey.Raw }}
            rangeKey: args.RangeKey,
            //TMPL {{- end }}
            billingMode: args.BillingMode,
        },
        { protect: args.protect }
    )
}
