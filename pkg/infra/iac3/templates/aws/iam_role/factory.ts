import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    AssumeRolePolicyDoc: string
    InlinePolicies: pulumi.Input<pulumi.Input<awsInputs.iam.RoleInlinePolicy>[]>
    ManagedPolicies: pulumi.Output<string>[]
    AwsManagedPolicies: string[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.iam.Role {
    return new aws.iam.Role(args.Name, {
        assumeRolePolicy: pulumi.jsonStringify(args.AssumeRolePolicyDoc),
        //TMPL {{- if .InlinePolicies }}
        inlinePolicies: args.InlinePolicies,
        //TMPL {{- end }}
        //TMPL {{- if or .ManagedPolicies .AwsManagedPolicies }}
        managedPolicyArns: [
            //TMPL {{- if .ManagedPolicies }}
            ...args.ManagedPolicies,
            //TMPL {{- end }}
            //TMPL {{- if .AwsManagedPolicies }}
            ...args.AwsManagedPolicies,
            //TMPL {{- end }}
        ],
        //TMPL {{- end }}
    })
}