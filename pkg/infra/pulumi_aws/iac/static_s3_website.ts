import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as mime from 'mime'
import * as fs from 'fs'
import * as path from 'path'

export const createStaticS3Website = (
    staticUnit: string,
    indexDocument: string,
    params
): pulumi.Output<string> => {
    // Create an S3 bucket

    const bucketArgs: aws.s3.BucketArgs = {}

    if (indexDocument != '' && !params.cloudFrontEnabled) {
        bucketArgs['website'] = {
            indexDocument: indexDocument,
        }
    }

    let siteBucket = new aws.s3.Bucket(`static-website-${staticUnit}`, bucketArgs)
    createAllObjects(staticUnit, siteBucket)

    if (params.cloudFrontEnabled) {
        // Generate Origin Access Identity to access the private s3 bucket.
        const originAccessIdentity = new aws.cloudfront.OriginAccessIdentity(
            'originAccessIdentity',
            {
                comment: 'this is needed to setup s3 polices and make s3 not public.',
            }
        )

        const bucketPolicy = new aws.s3.BucketPolicy('bucketPolicy', {
            bucket: siteBucket.id, // refer to the bucket created earlier
            policy: pulumi
                .all([originAccessIdentity.iamArn, siteBucket.arn])
                .apply(([oaiArn, bucketArn]) =>
                    JSON.stringify({
                        Version: '2012-10-17',
                        Statement: [
                            {
                                Effect: 'Allow',
                                Principal: {
                                    AWS: oaiArn,
                                }, // Only allow Cloudfront read access.
                                Action: ['s3:GetObject'],
                                Resource: [`${bucketArn}/*`], // Give Cloudfront access to the entire bucket.
                            },
                        ],
                    })
                ),
        })

        const distribution = new aws.cloudfront.Distribution(`cdn-static-${staticUnit}`, {
            origins: [
                {
                    domainName: siteBucket.bucketRegionalDomainName,
                    originId: siteBucket.arn,
                    s3OriginConfig: {
                        originAccessIdentity: originAccessIdentity.cloudfrontAccessIdentityPath,
                    },
                },
            ],
            enabled: true,
            viewerCertificate: {
                cloudfrontDefaultCertificate: true,
            },
            defaultCacheBehavior: {
                allowedMethods: ['DELETE', 'GET', 'HEAD', 'OPTIONS', 'PATCH', 'POST', 'PUT'],
                cachedMethods: ['GET', 'HEAD'],
                targetOriginId: siteBucket.arn,
                forwardedValues: {
                    queryString: false,
                    cookies: {
                        forward: 'none',
                    },
                },
                viewerProtocolPolicy: 'allow-all',
                minTtl: 0,
                defaultTtl: 3600,
                maxTtl: 86400,
            },
            restrictions: {
                geoRestriction: {
                    restrictionType: 'none',
                },
            },
            defaultRootObject: indexDocument,
        })

        return distribution.domainName
    } else {
        // Create an S3 Bucket Policy to allow public read of all objects in bucket
        // This reusable function can be pulled out into its own module
        function publicReadPolicyForBucket(bucketName) {
            return JSON.stringify({
                Version: '2012-10-17',
                Statement: [
                    {
                        Effect: 'Allow',
                        Principal: '*',
                        Action: ['s3:GetObject'],
                        Resource: [
                            `arn:aws:s3:::${bucketName}/*`, // policy refers to bucket name explicitly
                        ],
                    },
                ],
            })
        }

        // Set the access policy for the bucket so all objects are readable
        let bucketPolicy = new aws.s3.BucketPolicy('bucketPolicy', {
            bucket: siteBucket.bucket, // depends on siteBucket -- see explanation below
            policy: siteBucket.bucket.apply(publicReadPolicyForBucket),
            // transform the siteBucket.bucket output property -- see explanation below
        })

        return siteBucket.websiteEndpoint
    }
}

const createAllObjects = (staticUnit, siteBucket, prefixPath = '') => {
    // For each file in the directory, create an S3 object stored in `siteBucket`
    for (let item of fs.readdirSync(staticUnit)) {
        let filePath = path.join(staticUnit, item)
        let itemKey = prefixPath === '' ? item : `${prefixPath}/${item}`
        if (fs.statSync(filePath).isDirectory()) {
            createAllObjects(filePath, siteBucket, itemKey)
        } else {
            new aws.s3.BucketObject(`${staticUnit}-${item}`, {
                bucket: siteBucket,
                key: itemKey,
                source: new pulumi.asset.FileAsset(filePath), // use FileAsset to point to a file
                contentType: mime.getType(filePath) || undefined, // set the MIME type of the file
            })
        }
    }
}
