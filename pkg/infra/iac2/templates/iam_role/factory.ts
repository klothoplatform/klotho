import * as aws from '@pulumi/aws'
import * as awsInput from '@pulumi/aws/types/input'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    AssumeRolePolicyDoc: string
    InlinePolicies: pulumi.Input<pulumi.Input<input.iam.RoleInlinePolicy>[]>
    ManagedPolicies: pulumi.Output<string>[]
    AwsManagedPolicies: string[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.iam.Role {
    return new aws.iam.Role(args.Name, {
        assumeRolePolicy: pulumi.jsonStringify(args.AssumeRolePolicyDoc),
        //TMPL {{- if .InlinePolicies.Raw }}
        inlinePolicies: args.InlinePolicies,
        //TMPL {{- end }}
        //TMPL {{- if or .ManagedPolicies.Raw .AwsManagedPolicies.Raw }}
        managedPolicyArns: [
            //TMPL {{- if .ManagedPolicies.Raw }}
            ...args.ManagedPolicies,
            //TMPL {{- end }}
            //TMPL {{- if .AwsManagedPolicies.Raw }}
            ...args.AwsManagedPolicies,
            //TMPL {{- end }}
        ],
        //TMPL {{- end }}
    })
}
