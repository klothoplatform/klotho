import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecr.Repository {
    return new aws.ecr.Repository(args.Name, {
        imageScanningConfiguration: {
            scanOnPush: true,
        },
        imageTagMutability: 'MUTABLE',
        forceDelete: true,
        encryptionConfigurations: [{ encryptionType: 'KMS' }],
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
