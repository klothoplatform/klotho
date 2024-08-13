import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import * as path from 'path'
import * as pulumi from '@pulumi/pulumi'


const kloConfig = new pulumi.Config('klo')
const protect = kloConfig.getBoolean('protect') ?? false
const awsConfig = new pulumi.Config('aws')
const awsProfile = awsConfig.get('profile')
const accountId = pulumi.output(aws.getCallerIdentity({}))
const region = pulumi.output(aws.getRegion({}))

const my_bucket = aws.s3.Bucket.get("my-bucket", "preview(id=aws:s3_bucket:my-bucket)")
export const my_bucket_BucketName = my_bucket.id
const zip_func_function_executionrole = new aws.iam.Role("zip-func-function-ExecutionRole", {
        assumeRolePolicy: pulumi.jsonStringify({Statement: [{Action: ["sts:AssumeRole"], Effect: "Allow", Principal: {Service: ["lambda.amazonaws.com"]}}], Version: "2012-10-17"}),
        inlinePolicies: [
    {
        name: "my-bucket-policy",
        policy: pulumi.jsonStringify({Statement: [{Action: ["s3:DescribeJob", "s3:Get*", "s3:List*"], Effect: "Allow", Resource: [my_bucket.arn, pulumi.interpolate`${my_bucket.arn}/*`]}], Version: "2012-10-17"})
    },
],
        managedPolicyArns: [
            ...["arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"],
        ],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "zip-func-function-ExecutionRole"},
    })
const zip_func_function = new aws.lambda.Function(
        "zip-func-function",
        {
            handler: "handler.handler",
            runtime: "python3.12",
            code: new pulumi.asset.AssetArchive({
                ".": new pulumi.asset.FileArchive("/"),
            }),
            memorySize: 128,
            timeout: 3,
            role: zip_func_function_executionrole.arn,
            environment: {
                variables: {MY_BUCKET_BUCKET_NAME: my_bucket.id},
            },
            tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "zip-func-function"},
            loggingConfig: {
                logFormat: "Text",
            },
        },
        {
            dependsOn: [my_bucket, zip_func_function_executionrole],
        }
    )
const zip_func_function_log_group = new aws.cloudwatch.LogGroup("zip-func-function-log_group", {
        name: pulumi.interpolate`/aws/lambda/${zip_func_function.name}`,
        retentionInDays: 5,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "zip-func-function-log_group"},
    })

export const $outputs = {
	FunctionArn: zip_func_function.arn,
	FunctionName: zip_func_function.name,
}

export const $urns = {
	"aws:s3_bucket:my-bucket": (my_bucket as any).urn,
	"aws:iam_role:zip-func-function-ExecutionRole": (zip_func_function_executionrole as any).urn,
	"aws:lambda_function:zip-func-function": (zip_func_function as any).urn,
	"aws:log_group:zip-func-function-log_group": (zip_func_function_log_group as any).urn,
}
