const flatted = require('flatted')
const _ = require('lodash')
const path = require('path')
import { Readable } from 'stream'
import {
    S3Client,
    PutObjectCommand,
    GetObjectCommand,
    ListObjectsCommand,
    ListObjectsCommandInput,
    DeleteObjectCommand,
    HeadObjectCommand,
} from '@aws-sdk/client-s3'

const payloadBucketPhysicalName = process.env.KLOTHO_S3_PREFIX + '{{.PayloadsBucketName}}'
const targetRegion = process.env['AWS_TARGET_REGION']

const userBucketPath = '/files'

const s3Client = new S3Client({ region: targetRegion })

const streamToString = (stream: Readable): Promise<string> =>
    new Promise((resolve, reject) => {
        const chunks: Uint8Array[] = []
        stream.on('data', (chunk) => chunks.push(chunk))
        stream.on('error', reject)
        stream.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')))
    })

async function getCallParameters(paramKey, dispatcherMode) {
    let isEmitter = dispatcherMode === 'emitter' ? true : false
    try {
        const bucketParams = {
            Bucket: payloadBucketPhysicalName,
            Key: paramKey,
        }
        const result = await s3Client.send(new GetObjectCommand(bucketParams))

        let parameters = ''
        if (result.Body) {
            parameters = await streamToString(result.Body as Readable)
        }

        if (isEmitter && Array.isArray(parameters)) {
            // Emitters only have 1 parameter - the runtime saves an array, so we
            // normalize the parameter
            parameters = _.get(parameters, '[0]')
            if (Array.isArray(parameters)) {
                let paramPairs = _.toPairs(parameters)
                paramPairs = paramPairs.map((x) => {
                    if (_.get(x, '[1].type') == 'Buffer') {
                        return [x[0], Buffer.from(_.get(x, '[1].data'))]
                    } else {
                        return x
                    }
                })

                parameters = _.fromPairs(paramPairs)
            }
        }

        return parameters || {}
    } catch (e) {
        console.error(e)
        return
    }
}
exports.getCallParameters = getCallParameters

async function saveParametersToS3(paramsS3Key, params) {
    try {
        const bucketParams = {
            Bucket: payloadBucketPhysicalName,
            Key: paramsS3Key,
            Body: flatted.stringify(params),
        }
        await s3Client.send(new PutObjectCommand(bucketParams))
    } catch (e) {
        console.error(e)
        return
    }
}
exports.saveParametersToS3 = saveParametersToS3

async function s3_writeFile(...args) {
    const bucketParams = {
        Bucket: payloadBucketPhysicalName,
        Key: `${userBucketPath}/${args[0]}`,
        Body: args[1],
    }
    try {
        await s3Client.send(new PutObjectCommand(bucketParams))
        console.log('Successfully uploaded object: ' + bucketParams.Bucket + '/' + bucketParams.Key)
    } catch (err) {
        console.log('Error', err)
    }
}

async function s3_readFile(...args) {
    const bucketParams = {
        Bucket: payloadBucketPhysicalName,
        Key: `${userBucketPath}/${args[0]}`,
    }
    try {
        // Get the object from the Amazon S3 bucket. It is returned as a ReadableStream.
        const data = await s3Client.send(new GetObjectCommand(bucketParams))
        if (data.Body) {
            return await streamToString(data.Body as Readable)
        }
        return ''
    } catch (err) {
        console.log('Error', err)
    }
}

async function s3_readdir(path) {
    const bucketParams: ListObjectsCommandInput = {
        Bucket: payloadBucketPhysicalName,
        Prefix: `${userBucketPath}/${path}`,
    }

    try {
        const data = await s3Client.send(new ListObjectsCommand(bucketParams))
        if (data.Contents) {
            const objectKeys: string[] = data.Contents.map((c) => c.Key!)
            console.log('Success', objectKeys)
            return objectKeys
        }
    } catch (err) {
        console.log('Error', err)
    }
}

async function s3_exists(fpath) {
    const bucketParams = { Bucket: payloadBucketPhysicalName, Key: `${userBucketPath}/${path}` }
    try {
        const data = await s3Client.send(new HeadObjectCommand(bucketParams))
        console.log('Success. Object deleted.', data)
        return data // For unit tests.
    } catch (err) {
        console.log('Error', err)
    }
}

async function s3_deleteFile(fpath) {
    const bucketParams = { Bucket: payloadBucketPhysicalName, Key: `${userBucketPath}/${path}` }
    try {
        const data = await s3Client.send(new DeleteObjectCommand(bucketParams))
        console.log('Success. Object deleted.', data)
        return data // For unit tests.
    } catch (err) {
        console.log('Error', err)
    }
}

exports.fs = {
    writeFile: s3_writeFile,
    readFile: s3_readFile,
    readdir: s3_readdir,
    access: s3_exists,
    unlink: s3_deleteFile,
}
exports.fs.promises = exports.fs
