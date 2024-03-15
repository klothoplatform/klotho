import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Architecture: string
    ImageLocation: string
    RootDeviceName: string
    VirtualizationType: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

function create(args: Args): aws.ec2.Ami {
    return new aws.ec2.Ami(args.Name, {
        //TMPL {{- if .Architecture }}
        architecture: args.Architecture,
        //TMPL {{- end }}
        //TMPL {{- if .ImageLocation }}
        imageLocation: args.ImageLocation,
        //TMPL {{- end }}
        //TMPL {{- if .RootDeviceName }}
        rootDeviceName: args.RootDeviceName,
        //TMPL {{- end }}
        //TMPL {{- if .VirtualizationType }}
        virtualizationType: args.VirtualizationType,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ec2.Ami, args: Args) {
    return {
        Id: object.id,
    }
}