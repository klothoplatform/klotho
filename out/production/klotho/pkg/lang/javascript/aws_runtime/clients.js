const { S3Client } = require('@aws-sdk/client-s3')
const { LambdaClient } = require('@aws-sdk/client-lambda')
const { SecretsManagerClient } = require('@aws-sdk/client-secrets-manager')
const { SNSClient } = require('@aws-sdk/client-sns')
const { DynamoDBClient } = require('@aws-sdk/client-dynamodb')
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
    let secrets = new SecretsManagerClient(exports.AWSConfig)

    let s3 = new S3Client(exports.AWSConfig)
    let lambda = new LambdaClient(exports.AWSConfig)
    let sns = new SNSClient(exports.AWSConfig)
    let dynamo = new DynamoDBClient(exports.AWSConfig)

    s3 = AWSXRay.captureAWSv3Client(s3)
    lambda = AWSXRay.captureAWSv3Client(lambda)
    dynamo = AWSXRay.captureAWSv3Client(dynamo)
    secrets = AWSXRay.captureAWSv3Client(secrets)

    return { lambda, s3, secrets, sns, dynamo }
})()
