import {S3Client} from '@aws-sdk/client-s3'
import {LambdaClient} from '@aws-sdk/client-lambda'
import {SecretsManagerClient} from '@aws-sdk/client-secrets-manager'
import { SNSClient } from '@aws-sdk/client-sns'
import {DynamoDBClient} from '@aws-sdk/client-dynamodb'
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

    s3 = AWSXRay.captureAWSClient(s3)
    lambda = AWSXRay.captureAWSClient(lambda)
    dynamo = AWSXRay.captureAWSClient(dynamo)
    secrets = AWSXRay.captureAWSClient(secrets)

    return { lambda, s3, secrets, sns, dynamo }
})()
