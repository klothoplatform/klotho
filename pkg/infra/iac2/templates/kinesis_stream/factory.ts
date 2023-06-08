import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    RetentionPeriodHours: number
    ShardCount: number
    StreamEncryption: {
        Key: aws.kms.Key
        EncryptionType: string
    }
    StreamModeDetails: aws.types.input.kinesis.StreamStreamModeDetails
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.kinesis.Stream {
    return new aws.kinesis.Stream(args.Name, {
        retentionPeriod: args.RetentionPeriodHours,
        shardCount: args.ShardCount,
        //TMPL {{- if .StreamEncryption.Raw }}
        kmsKeyId: args.StreamEncryption.Key.id,
        encryptionType: args.StreamEncryption.EncryptionType,
        //TMPL {{- end }}
        streamModeDetails: args.StreamModeDetails,
        enforceConsumerDeletion: true,
    })
}
