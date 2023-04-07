import * as aws from '@pulumi/aws'
import * as fs from 'fs'

interface Args {
    SecretName: string
    Secret: aws.secretsmanager.Secret
    Path: string
    protect: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): void {
    return fs.readFile(args.Path, (err, data) => {
        if (err != null) {
            return
        }
        new aws.secretsmanager.SecretVersion(
            args.SecretName,
            {
                secretId: args.Secret.id,
                secretBinary: data.toString('base64'),
            },
            { protect: args.protect }
        )
    })
}
