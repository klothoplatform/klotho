import * as aws from '@pulumi/aws'
import * as fs from 'fs'

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
    protect: boolean
    CredentialsPath: string
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
            username: fs.readFileSync(args.CredentialsPath, 'utf-8').split('\n')[1].split('"')[3],
            password: fs.readFileSync(args.CredentialsPath, 'utf-8').split('\n')[2].split('"')[3],
            iamDatabaseAuthenticationEnabled: args.IamDatabaseAuthenticationEnabled,
            dbSubnetGroupName: args.SubnetGroup.name,
            vpcSecurityGroupIds: args.SecurityGroups.map((sg) => sg.id),
            skipFinalSnapshot: args.SkipFinalSnapshot,
            allocatedStorage: args.AllocatedStorage,
        },
        { protect: args.protect }
    )
}
