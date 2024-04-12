import * as pulumi from '@pulumi/pulumi'
import * as aws from '@pulumi/aws'
import { accountId, region, kloConfig } from '../../globals'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    SubnetGroup: aws.rds.SubnetGroup
    SecurityGroups: aws.ec2.SecurityGroup[]
    IamDatabaseAuthenticationEnabled: boolean
    DatabaseName: string
    Engine: string
    EngineVersion: string
    InstanceClass: string
    SkipFinalSnapshot: boolean
    AllocatedStorage: number
    Username: string
    Password: string
    protect: boolean
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.rds.Instance {
    return new aws.rds.Instance(
        args.Name,
        {
            instanceClass: args.InstanceClass,
            engine: args.Engine,
            engineVersion: args.EngineVersion,
            dbName: args.DatabaseName,
            username: kloConfig.requireSecret(`${args.Name}-username`),
            password: kloConfig.requireSecret(`${args.Name}-password`),
            iamDatabaseAuthenticationEnabled: args.IamDatabaseAuthenticationEnabled,
            dbSubnetGroupName: args.SubnetGroup.name,
            vpcSecurityGroupIds: args.SecurityGroups.map((sg) => sg.id),
            skipFinalSnapshot: args.SkipFinalSnapshot,
            allocatedStorage: args.AllocatedStorage,
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
        { protect: args.protect }
    )
}

function properties(object: aws.rds.Instance, args: Args) {
    return {
        Password: object.password.apply((pass) => pass!),
        Username: object.username,
        CredentialsSecretValue: pulumi.jsonStringify({
            username: object.username,
            password: object.password,
        }),
        RdsConnectionArn: pulumi.interpolate`arn:aws:rds-db:${region.name}:${accountId.accountId}:dbuser:${object.resourceId}/${object.username}`,
        Endpoint: object.endpoint,
        ConnectionString: pulumi.interpolate`${args.Engine}://${object.username}:${object.password}@${object.endpoint}/${args.DatabaseName}`,
        Host: object.endpoint.apply((endpoint) => endpoint.split(':')[0]),
        Port: object.endpoint.apply((endpoint) => endpoint.split(':')[1]),
    }
}

function infraExports(
    object: ReturnType<typeof create>,
    args: Args,
    props: ReturnType<typeof properties>
) {
    return {
        Address: object.address,
        Endpoint: object.endpoint,
    }
}

function importResource(args: Args): aws.rds.Instance {
    return aws.rds.Instance.get(args.Name, args.Arn)
}
