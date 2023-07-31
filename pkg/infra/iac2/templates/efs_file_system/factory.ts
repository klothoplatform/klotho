import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as awsInput from '@pulumi/aws/types/input'

interface Args {
    LifecyclePolicies: aws_input.efs.FileSystemLifecyclePolicy[]
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
            //TMPL {{- if .AvailabilityZoneName.Raw }}
            availabilityZoneName: args.AvailabilityZoneName,
            //TMPL {{- end }}
            //TMPL {{- if .CreationToken.Raw }}
            creationToken: args.CreationToken,
            //TMPL {{- end }}
            //TMPL {{- if .Encrypted.Raw }}
            encrypted: args.Encrypted,
            //TMPL {{- end }}
            //TMPL {{- if .KmsKey.Raw }}
            kmsKeyId: args.KmsKey?.arn,
            //TMPL {{- end }}
            //TMPL {{- if .LifecyclePolicies.Raw }}
            lifecyclePolicies: args.LifecyclePolicies,
            //TMPL {{- end }}
            //TMPL {{- if .PerformanceMode.Raw }}
            performanceMode: args.PerformanceMode,
            //TMPL {{- end }}
            //TMPL {{- if .ProvisionedThroughputInMibps.Raw }}
            provisionedThroughputInMibps: args.ProvisionedThroughputInMibps,
            //TMPL {{- end }}
            //TMPL {{- if .ThroughputMode.Raw }}
            throughputMode: args.ThroughputMode,
            //TMPL {{- end }}
        },
        {
            dependsOn: args.dependsOn,
        }
    )
}
