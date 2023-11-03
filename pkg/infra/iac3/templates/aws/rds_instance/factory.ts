import * as aws from '@pulumi/aws'
import * as fs from 'fs'
import { accountId, region, kloConfig } from '../../globals'

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
        },
        { protect: args.protect }
    )
}

function properties(object: aws.rds.Instance, args: Args) {
    return {
        Password: kloConfig.requireSecret(`${args.Name}-password`),
        Username: kloConfig.requireSecret(`${args.Name}-username`),
        CredentialsSecretValue: pulumi.jsonStringify({
            usename: object.username,
            password: object.password,
        }),
        RdsConnectionArn: pulumi.interpolate`arn:aws:rds-db:${region.name}:${accountId.accountId}:dbuser:${object.resourceId}/${object.username}`,
        HostName: object.endpoint,
    }
}
