import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'


const kloConfig = new pulumi.Config('klo')
const protect = kloConfig.getBoolean('protect') ?? false
const awsConfig = new pulumi.Config('aws')
const awsProfile = awsConfig.get('profile')
const accountId = pulumi.output(aws.getCallerIdentity({}))
const region = pulumi.output(aws.getRegion({}))

const my_bucket = new aws.s3.Bucket(
        "my-bucket",
        {
            forceDestroy: true,
            serverSideEncryptionConfiguration: {
                rule: {
                    applyServerSideEncryptionByDefault: {
                        sseAlgorithm: "aws:kms",
                    },
                    bucketKeyEnabled: true,
                },
            },
            website: {
                indexDocument: "index.html",
            },
            tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-bucket"},
        },
        { protect: protect }
    )
export const my_bucket_BucketName = my_bucket.bucket

export const $outputs = {
	BucketName: my_bucket.bucket,
	FixedValue: "some fixed value my-bucket",
}

export const $urns = {
	"aws:s3_bucket:my-bucket": (my_bucket as any).urn,
}
