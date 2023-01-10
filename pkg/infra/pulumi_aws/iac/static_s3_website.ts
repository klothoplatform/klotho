import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as mime from 'mime'
import * as fs from 'fs'
import * as path from 'path'
import { CloudCCLib } from '../deploylib'

export const createStaticS3Website = (
    staticUnit: string,
    indexDocument: string,
    contentDeliveryNetworkId: string,
    lib: CloudCCLib,
    appName: string
) => {
    // Create an S3 bucket

    lib.account.accountId.apply((accountId) => {
        const bucket = `${accountId}-${appName}-${staticUnit}`
        const bucketArgs: aws.s3.BucketArgs = { bucket }
        if (indexDocument != '' && contentDeliveryNetworkId == '') {
            bucketArgs['website'] = {
                indexDocument: indexDocument,
            }
        }

        let siteBucket = new aws.s3.Bucket(bucket, bucketArgs)
        lib.siteBuckets.set(staticUnit, siteBucket)
        createAllObjects(staticUnit, siteBucket)
    })
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
