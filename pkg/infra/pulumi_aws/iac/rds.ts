import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { hash as h, sanitized, validate } from './sanitization/sanitizer'
import AwsSanitizer from './sanitization/aws'
import { Resource, CloudCCLib, kloConfig } from '../deploylib'

export interface CreateInstanceAndProxyResult {
    rds: aws.rds.Instance
    proxy: aws.rds.Proxy
}

export interface RdsImport {
    dbInstanceIdentifier: string
    proxy: string
    dbName: string
}

export class RDS {
    constructor() {}

    static setupRDS(lib: CloudCCLib, orm: string, args: Partial<aws.rds.InstanceArgs>) {
        const rdsImports: { [key: string]: RdsImport } | undefined = kloConfig.getObject('rds')
        const config = new pulumi.Config()
        const dbName = sanitized(
            AwsSanitizer.RDS.engine.pg.database.nameValidation()
        )`${orm.toLowerCase()}`
        const username = config.require(`${dbName}_username`)
        const password = config.requireSecret(`${dbName}_password`)

        let rds
        let proxy

        if (rdsImports != undefined && rdsImports[orm] != undefined) {
            rds = aws.rds.Instance.get(dbName, rdsImports[orm].dbInstanceIdentifier)
            const proxyName = sanitized(AwsSanitizer.RDS.dbProxy.nameValidation())`${h(dbName)}`
            proxy = aws.rds.Proxy.get(proxyName, rdsImports[orm].proxy)
        } else {
            const result: CreateInstanceAndProxyResult = RDS.createInstanceAndProxy(
                lib,
                dbName,
                username,
                password,
                args
            )
            rds = result.rds
            proxy = result.proxy
        }

        if (rdsImports != undefined && rdsImports[orm]?.dbName != undefined) {
            const clients = lib.addConnectionString(
                orm,
                pulumi.interpolate`postgresql://${username}:${password}@${proxy.endpoint}:5432/${rdsImports[orm]?.dbName}`
            )
            RDS.addPermissions(lib, clients, rds, username)
        } else {
            const clients = lib.addConnectionString(
                orm,
                pulumi.interpolate`postgresql://${username}:${password}@${proxy.endpoint}:5432/${dbName}`
            )
            RDS.addPermissions(lib, clients, rds, username)
        }
    }

    static addPermissions(
        lib: CloudCCLib,
        clients: string[],
        rds: aws.rds.Instance,
        username: string
    ) {
        const resource = pulumi.interpolate`arn:aws:rds-db:${lib.region}:${lib.account.accountId}:dbuser:${rds.resourceId}/${username}`
        for (const client of clients) {
            lib.addPolicyStatementForName(lib.resourceIdToResource.get(client).title, {
                Effect: 'Allow',
                Action: ['rds-db:connect'],
                Resource: resource,
            })
        }
    }

    static createInstanceAndProxy(
        lib: CloudCCLib,
        dbName: string,
        username: string,
        password: pulumi.Output<string>,
        args: Partial<aws.rds.InstanceArgs>
    ): CreateInstanceAndProxyResult {
        if (!lib.subnetGroup) {
            const subnetGroupName = sanitized(AwsSanitizer.RDS.dbSubnetGroup.nameValidation())`${h(
                lib.name
            )}`
            lib.subnetGroup = new aws.rds.SubnetGroup(subnetGroupName, {
                subnetIds: lib.privateSubnetIds,
                tags: {
                    Name: 'Klotho DB subnet group',
                },
            })
        }

        // create the db resources
        validate(dbName, AwsSanitizer.RDS.instance.nameValidation())
        const rds = new aws.rds.Instance(
            dbName,
            {
                instanceClass: 'db.t4g.micro',
                ...args,
                engine: 'postgres',
                dbName: dbName,
                username: username,
                password: password,
                iamDatabaseAuthenticationEnabled: true,
                dbSubnetGroupName: lib.subnetGroup.name,
                vpcSecurityGroupIds: lib.sgs,
            },
            { protect: lib.protect }
        )

        // setup secrets for the proxy
        const secretName = `${dbName}_secret`
        const ssmSecretName = `${lib.name}-${secretName}`
        validate(ssmSecretName, AwsSanitizer.SecretsManager.secret.nameValidation())
        let rdsSecret = new aws.secretsmanager.Secret(`${secretName}`, {
            name: ssmSecretName,
            recoveryWindowInDays: 0,
        })

        const rdsSecretValues = {
            username: username,
            password: password,
            engine: 'postgres',
            host: rds.address,
            port: rds.port,
            dbname: dbName,
            dbInstanceIdentifier: rds.id,
            iamDatabaseAuthenticationEnabled: false,
        }

        const secret = new aws.secretsmanager.SecretVersion(`${secretName}`, {
            secretId: rdsSecret.id,
            secretString: pulumi.output(rdsSecretValues).apply((v) => JSON.stringify(v)),
        })

        lib.topology.topologyIconData.forEach((resource) => {
            if (resource.kind == Resource.secret) {
                lib.topology.topologyEdgeData.forEach((edge) => {
                    if (edge.target == resource.id) {
                        lib.addPolicyStatementForName(
                            lib.resourceIdToResource.get(edge.source).title,
                            {
                                Effect: 'Allow',
                                Action: ['secretsmanager:GetSecretValue'],
                                Resource: [secret.arn],
                            }
                        )
                    }
                })
            }
        })

        // prettier-ignore
        const ormRoleName = sanitized(AwsSanitizer.IAM.role.nameValidation())`${h(dbName)}-ormsecretrole`
        //setup role for proxy
        const role = new aws.iam.Role(ormRoleName, {
            assumeRolePolicy: {
                Version: '2012-10-17',
                Statement: [
                    {
                        Effect: 'Allow',
                        Principal: {
                            Service: 'rds.amazonaws.com',
                        },
                        Action: 'sts:AssumeRole',
                    },
                ],
            },
        })

        // prettier-ignore
        const ormPolicyName = sanitized(AwsSanitizer.IAM.policy.nameValidation())`${h(dbName)}-ormsecretpolicy`
        const policy = new aws.iam.Policy(ormPolicyName, {
            description: 'klotho orm secret policy',
            policy: {
                Version: '2012-10-17',
                Statement: [
                    {
                        Effect: 'Allow',
                        Action: 'secretsmanager:GetSecretValue',
                        Resource: [secret.arn],
                    },
                ],
            },
        })

        const attach = new aws.iam.RolePolicyAttachment(`${dbName}-ormattach`, {
            role: role.name,
            policyArn: policy.arn,
        })

        // setup the rds proxy
        const proxyName = sanitized(AwsSanitizer.RDS.dbProxy.nameValidation())`${h(dbName)}`
        const proxy = new aws.rds.Proxy(proxyName, {
            debugLogging: false,
            engineFamily: 'POSTGRESQL',
            idleClientTimeout: 1800,
            requireTls: false,
            roleArn: role.arn,
            vpcSecurityGroupIds: lib.sgs,
            vpcSubnetIds: lib.privateSubnetIds,
            auths: [
                {
                    authScheme: 'SECRETS',
                    description: 'use the secrets generated by klotho',
                    iamAuth: 'DISABLED',
                    secretArn: secret.arn,
                },
            ],
        })

        const proxyDefaultTargetGroup = new aws.rds.ProxyDefaultTargetGroup(`${dbName}`, {
            dbProxyName: proxy.name,
            connectionPoolConfig: {
                connectionBorrowTimeout: 120,
                maxConnectionsPercent: 100,
                maxIdleConnectionsPercent: 50,
            },
        })
        const proxyTarget = new aws.rds.ProxyTarget(`${dbName}`, {
            dbInstanceIdentifier: rds.id,
            dbProxyName: proxy.name,
            targetGroupName: proxyDefaultTargetGroup.name,
        })

        return {
            rds,
            proxy,
        }
    }
}
