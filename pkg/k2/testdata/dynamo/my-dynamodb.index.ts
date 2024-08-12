import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import * as pulumi from '@pulumi/pulumi'


const kloConfig = new pulumi.Config('klo')
const protect = kloConfig.getBoolean('protect') ?? false
const awsConfig = new pulumi.Config('aws')
const awsProfile = awsConfig.get('profile')
const accountId = pulumi.output(aws.getCallerIdentity({}))
const region = pulumi.output(aws.getRegion({}))

const my_dynamodb = new aws.dynamodb.Table(
        "my-dynamodb",
        {
            attributes: [
    {
        name: "id",
        type: "S"
    },
    {
        name: "data",
        type: "S"
    },
    {
        name: "status",
        type: "S"
    },
    {
        name: "timestamp",
        type: "N"
    },
]

,
            hashKey: "id",
            rangeKey: "data",
            billingMode: "PAY_PER_REQUEST",
            tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-dynamodb"},
            globalSecondaryIndexes: [{hashKey: "status", name: "StatusIndex", projectionType: "ALL"}],
            localSecondaryIndexes: [{name: "TimestampIndex", projectionType: "ALL", rangeKey: "timestamp"}],
        },
        { protect: protect }
    )

export const $outputs = {
	TableArn: my_dynamodb.arn,
	TableName: "my-dynamodb",
}

export const $urns = {
	"aws:dynamodb_table:my-dynamodb": (my_dynamodb as any).urn,
}
