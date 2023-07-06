import events = require('events')
import { PublishCommand, SNSClient } from '@aws-sdk/client-sns'
import {
    PutObjectCommand,
    GetObjectCommand,
    DeleteObjectCommand,
    S3Client,
} from '@aws-sdk/client-s3'
import { v4 as uuid } from 'uuid'
import * as crypto from 'crypto'
// @ts-ignore
import { addInflight } from './dispatcher'
import { Readable } from 'stream'

const payloadBucketPhysicalName = process.env.KLOTHO_PROXY_RESOURCE_NAME
const appName = '{{.AppName}}'

// The account-level ARN for sns. The topics must be account-wide unique
const { SNS_ARN_BASE } = process.env

export class Emitter extends events.EventEmitter {
    private client: SNSClient
    private s3: S3Client

    constructor(
        private path: string,
        private name: string,
        private id: string
    ) {
        super()

        this.client = new SNSClient({})
        this.s3 = new S3Client({})
    }

    override on(eventName: string | symbol, listener: (...args: any[]) => void): this {
        // wrap the listener and add it to the inflight promises in case the listener is an async function
        // otherwise a lambda will prematurely exist before the listener has run
        super.on(eventName, (...args: any[]) => {
            addInflight(listener(...args))
        })
        return this
    }

    /**
     * Must match the format used in deploylib
     */
    public topic(event: string): string {
        const topic = `${appName}_${this.id}_${event}`
        if (topic.length <= 256) {
            return topic
        }

        console.log('topic too long, hashing', { topic })
        const hash = crypto.createHash('sha256')
        hash.update(topic)
        return `${hash.digest('hex')}_${event}`
    }

    private async save(event: string, ...args: unknown[]): Promise<string> {
        const msgId = uuid()
        const key = `${this.path.replace(/[^0-9a-zA-Z_-]/, '-')}_${this.name}/${event}/${msgId}`

        await this.s3.send(
            new PutObjectCommand({
                Bucket: payloadBucketPhysicalName,
                Key: key,
                Body: JSON.stringify(args),
            })
        )

        return key
    }

    public async send(event: string, ...args: unknown[]) {
        const topic = this.topic(event)
        const arn = `${SNS_ARN_BASE}:${topic}`

        const payloadId = await this.save(event, ...args)

        const resp = await this.client.send(
            new PublishCommand({
                TopicArn: arn,
                Message: payloadId,
                MessageAttributes: {
                    Path: {
                        DataType: 'String',
                        StringValue: this.path,
                    },
                    Name: {
                        DataType: 'String',
                        StringValue: this.name,
                    },
                    Event: {
                        DataType: 'String',
                        StringValue: event,
                    },
                },
            })
        )

        console.info('Sent message', {
            event,
            topic,
            arn,
            payloadId,
            messageId: resp.MessageId,
        })
    }

    /**
     * @param record see https://docs.aws.amazon.com/lambda/latest/dg/with-sns.html
     */
    public async receive(record: any) {
        const { Message: payloadId, MessageAttributes: attribs } = record.Sns

        const eventName = attribs.Event.Value

        const obj = await this.s3.send(
            new GetObjectCommand({
                Bucket: payloadBucketPhysicalName,
                Key: payloadId,
            })
        )
        if (!obj.Body) return

        const argsStr = await streamToString(obj.Body as Readable)

        // TODO - would be nice to keep these around for a little for debugging/auditing purposes.
        const del = this.s3.send(
            new DeleteObjectCommand({
                Bucket: payloadBucketPhysicalName,
                Key: payloadId,
            })
        )
        addInflight(del)

        const args = JSON.parse(argsStr)

        this.emit(eventName, ...args)
    }
}

/**
 * see https://github.com/aws/aws-sdk-js-v3/issues/1877#issuecomment-755446927
 */
async function streamToString(stream: Readable): Promise<string> {
    return await new Promise((resolve, reject) => {
        const chunks: Uint8Array[] = []
        stream.on('data', (chunk) => chunks.push(chunk))
        stream.on('error', reject)
        stream.on('end', () => resolve(Buffer.concat(chunks).toString('utf-8')))
    })
}
