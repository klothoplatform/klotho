import * as aws from '@pulumi/aws'
import {kloConfig} from '../../globals'

interface Args {
    Name: string
    Secret: aws.secretsmanager.Secret
    Content: string
    Type: string
    protect: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.secretsmanager.SecretVersion {
    return new aws.secretsmanager.SecretVersion(
        args.Name,
        {
            secretId: args.Secret.id,
            //TMPL {{- if eq .Type "string" }}
            //TMPL {{- if .Content }}
            secretString: args.Content,
            //TMPL {{- else }}
            secretString: kloConfig.requireSecret(`${args.Name}-content`),
            //TMPL {{- end }}
            //TMPL {{- else }}
            //TMPL {{- if .Content }}
            secretBinary: args.Content,
            //TMPL {{- else }}
            secretBinary: kloConfig.requireSecret(`${args.Name}-content`),
            //TMPL {{- end }}
            //TMPL {{- end }}
        },
        {
            parent: args.Secret,
            protect: args.protect,
        }
    )   
}
