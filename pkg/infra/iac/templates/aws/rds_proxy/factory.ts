import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    DebugLogging: boolean
    EngineFamily: string
    IdleClientTimeout: number
    RequireTls: boolean
    Role: aws.iam.Role
    SecurityGroups: aws.ec2.SecurityGroup[]
    Subnets: aws.ec2.Subnet[]
    Auths: pulumi.Input<aws.types.input.rds.ProxyAuth>[]
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.rds.Proxy {
    return new aws.rds.Proxy(args.Name, {
        debugLogging: args.DebugLogging,
        engineFamily: args.EngineFamily,
        idleClientTimeout: args.IdleClientTimeout,
        requireTls: args.RequireTls,
        roleArn: args.Role.arn,
        vpcSecurityGroupIds: args.SecurityGroups.map((sg) => sg.id),
        vpcSubnetIds: args.Subnets.map((subnet) => subnet.id),
        auths: args.Auths,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.rds.Proxy, args: Args) {
    return {
        Endpoint: object.endpoint,
    }
}
