import * as aws from '@pulumi/aws'
import * as fs from 'fs'

interface Args {
    SecretName: string
    Secret: aws.secretsmanager.Secret
    Path: string
    Type: string
    protect: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.secretsmanager.SecretVersion {
    return new aws.secretsmanager.SecretVersion(
        args.SecretName,
        {
            secretId: args.Secret.id,
            //TMPL {{ if eq .Type.Raw "string" }}
            //TMPL secretString: fs.readFileSync(args.Path, 'utf-8').toString()
            //TMPL {{ else }}
            secretBinary: fs.readFileSync(args.Path, 'utf-8').toString('base64'),
            //TMPL {{ end }}
        },
        { protect: args.protect }
    )
}
