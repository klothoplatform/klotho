const S3 = require('aws-sdk/clients/s3')
const Lambda = require('aws-sdk/clients/lambda')
const Secrets = require('aws-sdk/clients/secretsmanager')
const SNS = require('aws-sdk/clients/sns')
const Dynamo = require('aws-sdk/clients/dynamodb')
const AWSXRay = require('aws-xray-sdk-core')

const endpoint = process.env['AWS_ENDPOINT'] ? `http://${process.env['AWS_ENDPOINT']}` : null
const targetRegion = process.env['AWS_TARGET_REGION']

exports.clients = (() => {
    let sharedAWSConfig = {
        region: targetRegion,
        s3ForcePathStyle: true,
        signatureVersion: 'v4',
    }

    if (endpoint) {
        sharedAWSConfig = {
            ...sharedAWSConfig,
            ...{
                accessKeyId: 'test',
                secretAccessKey: 'test',
                skipMetadataApiCheck: true,
                endpoint: endpoint,
            },
        }
    }

    let secrets = new Secrets(sharedAWSConfig)

    let s3 = new S3(sharedAWSConfig)
    let lambda = new Lambda(sharedAWSConfig)
    let sns = new SNS(sharedAWSConfig)
    let dynamo = new Dynamo(sharedAWSConfig)

    s3 = AWSXRay.captureAWSClient(s3)
    lambda = AWSXRay.captureAWSClient(lambda)
    dynamo = AWSXRay.captureAWSClient(dynamo)
    secrets = AWSXRay.captureAWSClient(secrets)

    return { lambda, s3, secrets, sns, dynamo }
})()
