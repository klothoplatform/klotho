import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    LaunchTemplateData: Record<string, pulumi.Input<any>>
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.LaunchTemplate {
    return new aws.ec2.LaunchTemplate(args.Name, {
        //TMPL {{- if .LaunchTemplateData.iamInstanceProfile }}
        //TMPL iamInstanceProfile: {{ .LaunchTemplateData.iamInstanceProfile }},
        //TMPL {{- end }}
        //TMPL {{- if .LaunchTemplateData.imageId }}
        //TMPL imageId: {{ .LaunchTemplateData.imageId }},
        //TMPL {{- else }}
        //TMPL imageId: aws.ec2.getAmi({
        //TMPL    filters: [
        //TMPL        {
        //TMPL            name: "name",
        //TMPL            values: ["amzn2-ami-hvm-*-x86_64-gp2"],
        //TMPL        },
        //TMPL    ],
        //TMPL    owners: ["137112412989"], // AWS account ID for Amazon AMIs
        //TMPL    mostRecent: true,
        //TMPL }).then(ami => ami.id),
        //TMPL {{- end }}
        //TMPL {{- if .LaunchTemplateData.instanceRequirements }}
        //TMPL instanceRequirements: {{ .LaunchTemplateData.instanceRequirements }},
        //TMPL {{- end }}
        //TMPL {{- if .LaunchTemplateData.instanceType }}
        //TMPL instanceType: args.LaunchTemplateData.instanceType,
        //TMPL {{- end }}
        //TMPL {{- if .LaunchTemplateData.securityGroupIds }}
        //TMPL securityGroupIds: {{ .LaunchTemplateData.securityGroupIds }},
        //TMPL {{- end }}
        //TMPL {{- if .LaunchTemplateData.userData }}
        //TMPL userData: {{ .LaunchTemplateData.userData }},
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ec2.LaunchTemplate, args: Args) {
    return {
        Arn: object.arn,
        Id: object.id,
    }
}
