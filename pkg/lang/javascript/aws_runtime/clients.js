const S3 = require('aws-sdk/clients/s3')
const Lambda = require('aws-sdk/clients/lambda')
const Secrets = require('aws-sdk/clients/secretsmanager')
const SNS = require('aws-sdk/clients/sns')
const Dynamo = require('aws-sdk/clients/dynamodb')
const AWSXRay = require('aws-xray-sdk-core')

const endpoint = process.env['AWS_ENDPOINT']
    ? `http://${process.env['AWS_ENDPOINT']}`
    : process.env['AWS_ENDPOINT_URL']
const targetRegion = process.env['AWS_TARGET_REGION']

exports.AWSConfig = {
    region: targetRegion,
    s3ForcePathStyle: true,
    signatureVersion: 'v4',
    ...(endpoint
        ? {
              accessKeyId: 'test',
              secretAccessKey: 'test',
              skipMetadataApiCheck: true,
              endpoint,
          }
        : {}),
}

exports.clients = (() => {
    let secrets = new Secrets(exports.AWSConfig)

    let s3 = new S3(exports.AWSConfig)
    let lambda = new Lambda(exports.AWSConfig)
    let sns = new SNS(exports.AWSConfig)
    let dynamo = new Dynamo(exports.AWSConfig)

    s3 = AWSXRay.captureAWSClient(s3)
    lambda = AWSXRay.captureAWSClient(lambda)
    dynamo = AWSXRay.captureAWSClient(dynamo)
    secrets = AWSXRay.captureAWSClient(secrets)

    return { lambda, s3, secrets, sns, dynamo }
})()
