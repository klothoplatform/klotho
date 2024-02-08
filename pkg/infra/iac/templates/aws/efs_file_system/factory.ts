import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as awsInputs from '@pulumi/aws/types/input'

interface Args {
    LifecyclePolicies: awsInputs.efs.FileSystemLifecyclePolicy[]
    ThroughputMode: string
    ProvisionedThroughputInMibps: number
    PerformanceMode: string
    Name: string
    AvailabilityZoneName?: string
    KmsKey?: aws.kms.Key
    Encrypted?: Promise<boolean> | pulumi.OutputInstance<boolean> | boolean
    CreationToken?: Promise<string> | pulumi.OutputInstance<string> | string
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.efs.FileSystem {
    return new aws.efs.FileSystem(
        args.Name,
        {
            //TMPL {{- if .AvailabilityZoneName }}
            availabilityZoneName: args.AvailabilityZoneName,
            //TMPL {{- end }}
            //TMPL {{- if .CreationToken }}
            creationToken: args.CreationToken,
            //TMPL {{- end }}
            //TMPL {{- if .Encrypted }}
            encrypted: args.Encrypted,
            //TMPL {{- end }}
            //TMPL {{- if .KmsKey }}
            kmsKeyId: args.KmsKey?.arn,
            //TMPL {{- end }}
            //TMPL {{- if .LifecyclePolicies }}
            lifecyclePolicies: args.LifecyclePolicies,
            //TMPL {{- end }}
            //TMPL {{- if .PerformanceMode }}
            performanceMode: args.PerformanceMode,
            //TMPL {{- end }}
            //TMPL {{- if .ProvisionedThroughputInMibps }}
            provisionedThroughputInMibps: args.ProvisionedThroughputInMibps,
            //TMPL {{- end }}
            //TMPL {{- if .ThroughputMode }}
            throughputMode: args.ThroughputMode,
            //TMPL {{- end }}
        },
        {
            dependsOn: args.dependsOn,
        }
    )
}

function properties(object: aws.efs.FileSystem, args: Args) {
    return {
        Id: object.id,
        Arn: object.arn,
    }
}
