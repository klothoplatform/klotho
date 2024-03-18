import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Role: aws.iam.Role
    Tags: ModelCaseWrapper<Record<string, string>>
}

function create(args: Args): aws.iam.InstanceProfile {
    return new aws.iam.InstanceProfile(args.Name, {
        role: args.Role,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ecs.Cluster, args: Args) {
    return {
        Arn: object.arn,
    }
}
