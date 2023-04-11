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
function create(args: Args): void {
    return fs.readFile(args.Path, (err, data) => {
        if (err != null) {
            return
        }
        new aws.secretsmanager.SecretVersion(
            args.SecretName,
            {
                secretId: args.Secret.id,
                //TMPL {{ if eq .Type.Raw "string" }}
                //TMPL secretString: data.toString()
                //TMPL {{ else }}
                secretBinary: data.toString('base64'),
                //TMPL {{ end }}
            },
            { protect: args.protect }
        )
    })
}
