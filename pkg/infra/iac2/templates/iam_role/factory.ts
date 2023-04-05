import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    AssumeRolePolicyDoc: string
    InlinePolicy: aws.iam.PolicyDocument
    ManagedPolicies: pulumi.Output<string>[]
    AwsManagedPolicies: string[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.iam.Role {
    return new aws.iam.Role(args.Name, {
        assumeRolePolicy: args.AssumeRolePolicyDoc,
        //TMPL {{ if .InlinePolicy.Raw }}
        inlinePolicies: [
            {
                name: args.Name,
                policy: JSON.stringify(args.InlinePolicy),
            },
        ],
        //TMPLE {{ end }}
        managedPolicyArns: [...args.ManagedPolicies, ...args.AwsManagedPolicies],
    })
}
