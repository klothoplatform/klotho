import * as aws from '@pulumi/aws'
import * as fs from 'fs'

interface Args {
    Name: string
    Secret: aws.secretsmanager.Secret
    Path: string
    Type: string
    protect: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): void {
    if (fs.existsSync(args.Path)) {
        new aws.secretsmanager.SecretVersion(
            args.Name,
            {
                secretId: args.Secret.id,
                //TMPL {{- if eq .Type.Raw "string" }}
                secretString: fs.readFileSync(args.Path, 'utf-8').toString(),
                //TMPL {{- else }}
                //TMPL secretBinary: fs.readFileSync({{ .Path.Parse }}, 'base64').toString(),
                //TMPL {{- end }}
            },
            {
                parent: args.Secret,
                protect: args.protect,
            }
        )
    }
}
