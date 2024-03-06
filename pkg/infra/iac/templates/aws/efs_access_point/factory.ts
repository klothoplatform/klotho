import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    FileSystem: aws.efs.FileSystem
    RootDirectory?: awsInputs.efs.AccessPointRootDirectory
    PosixUser?: awsInputs.efs.AccessPointPosixUser
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.efs.AccessPoint {
    return new aws.efs.AccessPoint(args.Name, {
        //TMPL {{- if .RootDirectory }}
        fileSystemId: args.FileSystem.id,
        //TMPL {{- end }}
        //TMPL {{- if .PosixUser }}
        posixUser: args.PosixUser,
        //TMPL {{- end }}
        //TMPL {{- if .RootDirectory }}
        rootDirectory: args.RootDirectory,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
