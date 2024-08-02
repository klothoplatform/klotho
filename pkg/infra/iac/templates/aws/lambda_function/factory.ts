import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'
import * as path from "path";

interface Args {
    Name: string
    Image: string
    ExecutionRole: aws.iam.Role
    EnvironmentVariables: ModelCaseWrapper<Record<string, pulumi.Output<string>>>
    Subnets: aws.ec2.Subnet[]
    SecurityGroups: aws.ec2.SecurityGroup[]
    MemorySize: pulumi.Input<number>
    Timeout: pulumi.Input<number>
    EfsAccessPoint: aws.efs.AccessPoint
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
    Tags: ModelCaseWrapper<Record<string, string>>
    Code: string
    Handler: string
    Runtime: string
    S3Bucket: string
    S3Key: string
    S3ObjectVersion: string
    Id: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lambda.Function {
    return new aws.lambda.Function(
        args.Name,
        {
            //TMPL {{- if .Code }}
            handler: args.Handler,
            runtime: args.Runtime,
            //TMPL {{- if matches `^https?:.+` .Code }}
            code: new pulumi.asset.RemoteArchive(args.Code),
            //TMPL {{- else if matches `[^\\/]+\.[\w]+$` .Code  }}
            //TMPL code: new pulumi.asset.FileArchive(args.Code),
            //TMPL {{- else }}
            //TMPL code: new pulumi.asset.AssetArchive({
            //TMPL     ".": new pulumi.asset.FileArchive(args.Code),
            //TMPL }),
            //TMPL {{- end }}
            //TMPL {{- else if .S3Bucket }}
            s3Bucket: args.S3Bucket,
            s3Key: args.S3Key,
            s3ObjectVersion: args.S3ObjectVersion,
            //TMPL {{- else if .Image }}
            packageType: 'Image',
            imageUri: args.Image,
            //TMPL {{- end }}
            //TMPL {{- if .MemorySize }}
            memorySize: args.MemorySize,
            //TMPL {{- end }}
            //TMPL {{- if .Timeout }}
            timeout: args.Timeout,
            //TMPL {{- end }}
            role: args.ExecutionRole.arn,
            //TMPL {{- if .EfsAccessPoint }}
            fileSystemConfig: {
                arn: args.EfsAccessPoint.arn,
                localMountPath: args.EfsAccessPoint.rootDirectory.path,
            },
            //TMPL {{- end }}
            //TMPL {{- if and .SecurityGroups .Subnets }}
            vpcConfig: {
                securityGroupIds: args.SecurityGroups.map((sg) => sg.id),
                subnetIds: args.Subnets.map((subnet) => subnet.id),
            },
            //TMPL {{- end }}
            //TMPL {{- if .EnvironmentVariables }}
            environment: {
                variables: args.EnvironmentVariables,
            },
            //TMPL {{- end }}
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
        {
            dependsOn: args.dependsOn,
        }
    )
}

function properties(object: aws.lambda.Function, args: Args) {
    return {
        LambdaIntegrationUri: object.invokeArn,
        Arn: object.arn,
        FunctionName: object.name,
        DefaultLogGroup: pulumi.interpolate`/aws/lambda/${object.name}`,
    }
}

function importResource(args: Args): aws.lambda.Function {
    return aws.lambda.Function.get(args.Name, args.Id)
}
