import * as aws from '@pulumi/aws'
import * as awsInput from '@pulumi/aws/types/input'
interface Args {
    Name: string
    FileSystem: aws.efs.FileSystem
    RootDirectory?: aws_input.efs.AccessPointRootDirectory
    PosixUser?: aws_input.efs.AccessPointPosixUser
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.efs.AccessPoint {
    return new aws.efs.AccessPoint(args.Name, {
        //TMPL {{- if .RootDirectory.Raw }}
        fileSystemId: args.FileSystem.id,
        //TMPL {{- end }}
        //TMPL {{- if .PosixUser.Raw }}
        posixUser: args.PosixUser,
        //TMPL {{- end }}
        //TMPL {{- if .RootDirectory.Raw }}
        rootDirectory: args.RootDirectory,
        //TMPL {{- end }}
    })
}
