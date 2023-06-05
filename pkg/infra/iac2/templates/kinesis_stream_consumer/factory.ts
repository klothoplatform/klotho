import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    ConsumerName: string
    Stream: aws.kinesis.Stream
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.kinesis.StreamConsumer {
    return new aws.kinesis.StreamConsumer(args.Name, {
        streamArn: args.Stream.arn,
        name: args.ConsumerName,
    })
}
