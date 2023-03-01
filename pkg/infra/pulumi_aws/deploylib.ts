import * as aws from '@pulumi/aws'
import { Region } from '@pulumi/aws'
import * as awsx from '@pulumi/awsx'

import * as pulumi from '@pulumi/pulumi'
import * as sha256 from 'simple-sha256'
import * as fs from 'fs'
import * as requestRetry from 'requestretry'
import * as crypto from 'crypto'
import { setupElasticacheCluster } from './iac/elasticache'
import * as analytics from './iac/analytics'

import { hash as h, sanitized, validate } from './iac/sanitization/sanitizer'
import { LoadBalancerPlugin } from './iac/load_balancing'
import { DefaultEksClusterOptions, Eks, EksExecUnit, HelmChart } from './iac/eks'
import { setupMemoryDbCluster } from './iac/memorydb'
import AwsSanitizer from './iac/sanitization/aws'

export enum Resource {
    exec_unit = 'exec_unit',
    static_unit = 'static_unit',
    gateway = 'gateway',
    kv = 'persist_kv',
    fs = 'persist_fs',
    secret = 'persist_secret',
    orm = 'persist_orm',
    redis_node = 'persist_redis_node',
    redis_cluster = 'persist_redis_cluster',
    pubsub = 'pubsub',
    internal = 'internal',
    config = 'config',
}

export interface ResourceKey {
    Kind: string
    Name: string
}

interface ResourceInfo {
    id: string
    urn: string
    kind: string
    type: string
    url: string
}

export interface TopologyData {
    topologyIconData: TopologyIconData[]
    topologyEdgeData: TopologyEdgeData[]
}

export interface TopologyIconData {
    id: string
    title: string
    image: string
    kind: string
    type: string
}

export interface TopologyEdgeData {
    source: string
    target: string
}

export const kloConfig: pulumi.Config = new pulumi.Config('klo')

export class CloudCCLib {
    secrets = new Map<string, aws.secretsmanager.Secret>()

    resourceIdToResource = new Map<string, any>()
    sharedPolicyStatements: aws.iam.PolicyStatement[] = []
    execUnitToFunctions = new Map<string, aws.lambda.Function>()
    execUnitToRole = new Map<string, aws.iam.Role>()
    execUnitToPolicyStatements = new Map<string, aws.iam.PolicyStatement[]>()
    execUnitToImage = new Map<string, pulumi.Output<string>>()

    gatewayToUrl = new Map<string, pulumi.Output<string>>()
    siteBuckets = new Map<string, aws.s3.Bucket>()
    buckets = new Map<string, aws.s3.Bucket>()

    topologySpecOutputs: pulumi.Output<ResourceInfo>[] = []
    connectionString = new Map<string, pulumi.Output<string>>()

    klothoVPC: awsx.ec2.Vpc
    publicSubnetIds: Promise<pulumi.Output<string>[]>
    privateSubnetIds: Promise<pulumi.Output<string>[]>
    sgs: pulumi.Output<string>[] = []
    subnetGroup: aws.rds.SubnetGroup

    snsTopics = new Map<string, aws.sns.Topic>()

    account: pulumi.Output<aws.GetCallerIdentityResult>
    cluster: awsx.ecs.Cluster
    privateDnsNamespace: aws.servicediscovery.PrivateDnsNamespace

    eks: Eks

    protect = kloConfig.getBoolean('protect') ?? false
    execUnitToNlb = new Map<string, awsx.lb.NetworkLoadBalancer>()
    execUnitToVpcLink = new Map<string, aws.apigateway.VpcLink>()

    constructor(
        private sharedRepo: awsx.ecr.Repository,
        public readonly stage: string,
        public readonly region: Region,
        public readonly name: string,
        private namespace: string,
        private datadogEnabled: boolean,
        public readonly topology: TopologyData,
        private createVPC: boolean
    ) {
        this.account = pulumi.output(aws.getCallerIdentity({}))
        // These are CloudCompiler specific components that are required for it's operation
        this.generateResourceMap()
        if (this.createVPC) {
            this.getVpcSgSubnets()
        }
        this.addSharedPolicyStatement({
            Effect: 'Allow',
            Action: ['cloudwatch:PutMetricData'],
            Resource: '*',
            Condition: {
                StringEquals: {
                    'cloudwatch:namespace': this.namespace,
                },
            },
        })
    }

    getVpcSgSubnets() {
        interface VPC {
            id?: string
            sgId?: string
            publicSubnetIds?: string[]
            privateSubnetIds?: string[]
        }

        const klothoVPC = kloConfig.getObject<VPC>('vpc')
        if (klothoVPC == undefined) {
            this.createVpcSgSubnets()
            this.createVpcEndpoints()
            return
        }
        if (
            klothoVPC.id == undefined ||
            klothoVPC.sgId == undefined ||
            klothoVPC.privateSubnetIds == undefined
        ) {
            throw new Error(
                'id, sgId, and privateSubnetIds[] must all be valid and specified in your pulumi config.'
            )
        }

        this.klothoVPC = awsx.ec2.Vpc.fromExistingIds(this.name, {
            vpcId: klothoVPC.id,
            privateSubnetIds: klothoVPC.privateSubnetIds,
            publicSubnetIds: klothoVPC.publicSubnetIds,
        })

        this.publicSubnetIds = this.klothoVPC.publicSubnetIds
        this.privateSubnetIds = this.klothoVPC.privateSubnetIds

        const klothoSG = awsx.ec2.SecurityGroup.fromExistingId(this.name, klothoVPC.sgId)
        this.sgs = new Array(klothoSG.id)
    }

    createVpcSgSubnets() {
        this.klothoVPC = new awsx.ec2.Vpc(this.name, {
            cidrBlock: '10.0.0.0/16',
            enableDnsHostnames: true,
            enableDnsSupport: true,
            numberOfAvailabilityZones: 2,
            subnets: [{ type: 'public' }, { type: 'private' }],
        })

        this.publicSubnetIds = this.klothoVPC.publicSubnetIds
        this.privateSubnetIds = this.klothoVPC.privateSubnetIds

        const sgName = sanitized(AwsSanitizer.EC2.vpc.securityGroup.nameValidation())`${h(
            this.name
        )}`
        const klothoSG = new aws.ec2.SecurityGroup(sgName, {
            name: sgName,
            vpcId: this.klothoVPC.id,
            egress: [
                {
                    cidrBlocks: ['0.0.0.0/0'],
                    description: 'Allows all outbound IPv4 traffic.',
                    fromPort: 0,
                    protocol: '-1',
                    toPort: 0,
                },
            ],
            ingress: [
                {
                    description:
                        'Allows inbound traffic from network interfaces and instances that are assigned to the same security group.',
                    fromPort: 0,
                    protocol: '-1',
                    self: true,
                    toPort: 0,
                },
                {
                    description: 'For EKS control plane',
                    cidrBlocks: ['0.0.0.0/0'],
                    fromPort: 9443,
                    protocol: 'TCP',
                    self: true,
                    toPort: 9443,
                },
            ],
        })
        this.sgs = new Array(klothoSG.id)

        pulumi.output(this.klothoVPC.privateSubnets).apply((ps) => {
            const cidrBlocks: any = ps.map((subnet) => subnet.subnet.cidrBlock)
            new aws.ec2.SecurityGroupRule(`${this.name}-private-ingress`, {
                type: 'ingress',
                cidrBlocks: cidrBlocks,
                fromPort: 0,
                protocol: '-1',
                toPort: 0,
                securityGroupId: klothoSG.id,
            })
        })

        pulumi.output(this.klothoVPC.publicSubnets).apply((ps) => {
            const cidrBlocks: any = ps.map((subnet) => subnet.subnet.cidrBlock)
            new aws.ec2.SecurityGroupRule(`${this.name}-public-ingress`, {
                type: 'ingress',
                cidrBlocks: cidrBlocks,
                fromPort: 0,
                protocol: '-1',
                toPort: 0,
                securityGroupId: klothoSG.id,
            })
        })
    }

    // there is currently no way to handle an exception of a resource doesn't exist, so this
    // actually only creates vpc endpoints.
    getOrCreateVpcEndpoint(
        name: string,
        type: string,
        awsServiceName: string,
        subnetIds: pulumi.Output<string>[],
        routeTableIds: pulumi.Output<string>[]
    ) {
        if (type == 'Interface') {
            const endpoint = new aws.ec2.VpcEndpoint(name, {
                vpcId: this.klothoVPC.id,
                subnetIds: subnetIds,
                securityGroupIds: this.sgs,
                serviceName: awsServiceName,
                vpcEndpointType: 'Interface',
                privateDnsEnabled: true,
            })
        } else if (type == 'Gateway') {
            const endpoint = new aws.ec2.VpcEndpoint(name, {
                vpcId: this.klothoVPC.id,
                serviceName: awsServiceName,
                routeTableIds: routeTableIds,
            })
        }
    }

    createVpcEndpoints() {
        this.klothoVPC.privateSubnets.then((subnets) => {
            const subnetIds: pulumi.Output<string>[] = subnets.map((x) => x.id)
            const routeTableIds = subnets.map((x) => x.routeTable!.id)

            for (const svc of ['lambda', 'sqs', 'sns', 'secretsmanager']) {
                this.getOrCreateVpcEndpoint(
                    `${svc}VpcEndpoint`,
                    'Interface',
                    `com.amazonaws.${this.region}.${svc}`,
                    subnetIds,
                    routeTableIds
                )
            }

            for (const svc of ['dynamodb', 's3']) {
                this.getOrCreateVpcEndpoint(
                    `${svc}VpcEndpoint`,
                    'Gateway',
                    `com.amazonaws.${this.region}.${svc}`,
                    subnetIds,
                    routeTableIds
                )
            }
        })
    }

    generateResourceMap() {
        this.topology.topologyIconData.forEach((r) => {
            this.resourceIdToResource.set(r.id, r)
        })
    }

    addSharedPolicyStatement(statement: aws.iam.PolicyStatement) {
        this.sharedPolicyStatements.push(statement)
    }

    addPolicyStatementForName(name: string, statement: aws.iam.PolicyStatement) {
        if (this.execUnitToPolicyStatements.has(name)) {
            let statements = this.execUnitToPolicyStatements.get(name)
            statements!.push(statement)
        } else {
            this.execUnitToPolicyStatements.set(name, [statement])
        }
    }

    // make sure this is called last so all resource generation has a chance to add to the policy statements
    createPolicy() {
        for (const [physicalName, role] of this.execUnitToRole.entries()) {
            const combinedPolicyStatements = new Set<aws.iam.PolicyStatement>([
                ...this.sharedPolicyStatements,
            ])
            if (this.execUnitToPolicyStatements.has(physicalName)) {
                this.execUnitToPolicyStatements
                    .get(physicalName)!
                    .forEach((item) => combinedPolicyStatements.add(item))
            }
            if (combinedPolicyStatements.size > 0) {
                const policyName = sanitized(AwsSanitizer.IAM.policy.nameValidation())`${h(
                    this.name
                )}-${h(physicalName)}-exec`
                const policy = new aws.iam.Policy(
                    policyName,
                    {
                        policy: {
                            Version: '2012-10-17',
                            Statement: Array.from(combinedPolicyStatements),
                        },
                    },
                    { parent: role }
                )
                new aws.iam.RolePolicyAttachment(
                    policyName,
                    {
                        role: role,
                        policyArn: policy.arn,
                    },
                    { parent: role }
                )
            }
        }
    }

    createImage(execUnitName: string, dockerfilePath: string) {
        const image = this.sharedRepo.buildAndPushImage({
            context: `./${execUnitName}`,
            dockerfile: `./${execUnitName}/${dockerfilePath}`,
            extraOptions: ['--platform', 'linux/amd64', '--quiet'],
        })
        this.execUnitToImage.set(execUnitName, image)
    }

    createDockerAppRunner(execUnitName, envVars: any) {
        const instanceRole = this.createRoleForName(execUnitName)

        this.addPolicyStatementForName(execUnitName, {
            Effect: 'Allow',
            Action: ['apprunner:ListServices'],
            Resource: ['*'],
        })

        const roleName = sanitized(AwsSanitizer.IAM.role.nameValidation())`${h(this.name)}-${h(
            execUnitName
        )}-ar-access-role`
        const accessRole = new aws.iam.Role(roleName, {
            assumeRolePolicy: {
                Version: '2012-10-17',
                Statement: [
                    {
                        Effect: 'Allow',
                        Principal: {
                            Service: 'build.apprunner.amazonaws.com',
                        },
                        Action: 'sts:AssumeRole',
                    },
                ],
            },
        })

        const policy = new aws.iam.Policy(
            sanitized(AwsSanitizer.IAM.policy.nameValidation())`${h(this.name)}-${h(
                execUnitName
            )}-ar-access-policy`,
            {
                description: 'Role to grant AppRunner service access to ECR',
                policy: {
                    Version: '2012-10-17',
                    Statement: [
                        {
                            Effect: 'Allow',
                            Action: [
                                'ecr:GetDownloadUrlForLayer',
                                'ecr:BatchGetImage',
                                'ecr:DescribeImages',
                                'ecr:GetAuthorizationToken',
                                'ecr:BatchCheckLayerAvailability',
                            ],
                            Resource: ['*'],
                        },
                    ],
                },
            },
            {
                parent: accessRole,
            }
        )

        const attach = new aws.iam.RolePolicyAttachment(
            `${execUnitName}-ar-access-attach`,
            {
                role: accessRole.name,
                policyArn: policy.arn,
            },
            {
                parent: accessRole,
            }
        )

        accessRole.arn.apply(async () => {
            await new Promise((f) => setTimeout(f, 8000))
        })

        const additionalEnvVars: { [key: string]: pulumi.Input<string> } =
            this.generateExecUnitEnvVars(execUnitName, envVars)

        const logGroupName = sanitized(
            AwsSanitizer.CloudWatch.logGroup.nameValidation()
        )`/aws/apprunner/${h(this.name)}-${h(execUnitName)}-apprunner`
        let cloudwatchGroup = new aws.cloudwatch.LogGroup(`${this.name}-${execUnitName}-lg`, {
            name: logGroupName,
            retentionInDays: 1,
        })

        let isProxied = false
        this.topology.topologyEdgeData.forEach((edge) => {
            if (this.resourceIdToResource.get(edge.target)?.title == execUnitName) {
                if (this.resourceIdToResource.get(edge.source)?.kind == Resource.exec_unit) {
                    isProxied = true
                }
            }
        })

        const serviceName = sanitized(AwsSanitizer.AppRunner.service.nameValidation())`${h(
            this.name
        )}-${h(execUnitName)}-apprunner`
        const service = new aws.apprunner.Service(serviceName, {
            serviceName: serviceName,
            sourceConfiguration: {
                autoDeploymentsEnabled: false,
                authenticationConfiguration: {
                    accessRoleArn: accessRole.arn,
                },
                imageRepository: {
                    imageConfiguration: {
                        port: isProxied ? '3001' : '3000',
                        runtimeEnvironmentVariables: additionalEnvVars,
                    },
                    imageIdentifier: this.execUnitToImage.get(execUnitName)!,
                    imageRepositoryType: 'ECR',
                },
            },
            tags: {
                Name: serviceName,
                App: this.name,
            },
            instanceConfiguration: {
                instanceRoleArn: instanceRole.arn,
            },
        })

        if (!isProxied) {
            let resp = {}
            resp[`${execUnitName}`] = pulumi.interpolate`https://${service.serviceUrl}`
            return resp
        }
    }

    createDockerLambda(
        execUnitName,
        baseArgs: Partial<aws.lambda.FunctionArgs>,
        network_placement: 'private' | 'public',
        env_vars?: any[]
    ) {
        const image = this.execUnitToImage.get(execUnitName)!

        const subnetIds =
            network_placement === 'public' ? this.publicSubnetIds : this.privateSubnetIds

        const lambdaRole = this.createRoleForName(execUnitName)
        const lambdaName = sanitized(AwsSanitizer.Lambda.lambdaFunction.nameValidation())`${h(
            this.name
        )}-${h(execUnitName)}`

        const lambdaConfig: aws.lambda.FunctionArgs = {
            ...baseArgs,
            packageType: 'Image',
            imageUri: image,
            role: lambdaRole.arn,
            name: lambdaName,
            tags: {
                env: 'production',
                service: execUnitName,
            },
            vpcConfig: this.createVPC
                ? {
                      securityGroupIds: this.sgs,
                      subnetIds,
                  }
                : undefined,
        }
        if (this.datadogEnabled) {
            lambdaConfig.tracingConfig = {
                mode: 'Active',
            }
        }

        let cloudwatchGroup = new aws.cloudwatch.LogGroup(`${execUnitName}-function-api-lg`, {
            name: pulumi.interpolate`/aws/lambda/${lambdaConfig.name}`,
            retentionInDays: 1,
        })

        const additionalEnvVars = this.generateExecUnitEnvVars(execUnitName, env_vars)

        if (lambdaConfig.environment != null) {
            lambdaConfig.environment = pulumi.output(lambdaConfig.environment).apply((env) => ({
                variables: { ...env.variables, ...additionalEnvVars },
            }))
        } else {
            lambdaConfig.environment = { variables: additionalEnvVars }
        }
        let createdFunction = new aws.lambda.Function(execUnitName, lambdaConfig, {
            dependsOn: [cloudwatchGroup],
        })

        this.topologySpecOutputs.push(
            pulumi.all([createdFunction.id, createdFunction.urn]).apply(([id, urn]) => ({
                id: id,
                urn: urn,
                kind: '', // TODO
                type: 'AWS Lambda',
                url: `https://console.aws.amazon.com/lambda/home?region=${this.region}#/functions/${id}?tab=code`,
            }))
        )

        this.execUnitToFunctions.set(execUnitName, createdFunction)
        return createdFunction
    }

    getEnvVarForDependency(v: any): [string, pulumi.Input<string>] {
        switch (v.Kind) {
            case Resource.orm:
                const connStr = this.connectionString.get(`${v.ResourceID}_${v.Kind}`)!
                return [v.Name, connStr]
                break
            case Resource.redis_node:
            // intentional fall-through: redis-node and redis_cluster get configured the same way
            case Resource.redis_cluster:
                if (v.Value === 'host') {
                    return [v.Name, this.connectionString.get(`${v.ResourceID}_${v.Kind}_host`)!]
                } else if (v.Value === 'port') {
                    return [v.Name, this.connectionString.get(`${v.ResourceID}_${v.Kind}_port`)!]
                }
                break
            case Resource.secret:
                const secret: aws.secretsmanager.Secret = this.secrets.get(v.ResourceID)!
                return [v.Name, secret.name]
            case Resource.fs:
                const bucket: aws.s3.Bucket = this.buckets.get(v.ResourceID)!
                return [v.Name, bucket.bucket]
            case Resource.internal:
                if (v.ResourceID === 'InternalKlothoPayloads') {
                    const bucket: aws.s3.Bucket = this.buckets.get(v.ResourceID)!
                    return [v.Name, bucket.bucket]
                }
                break
            case Resource.config:
                if (v.Value == 'secret_name') {
                    const secret: aws.secretsmanager.Secret = this.secrets.get(v.ResourceID)!
                    return [v.Name, secret.name]
                }
                break
            default:
                throw new Error('unsupported kind')
        }
        return ['', '']
    }

    generateExecUnitEnvVars(
        execUnitName: string,
        env_vars?: any[]
    ): { [key: string]: pulumi.Input<string> } {
        const additionalEnvVars: { [key: string]: pulumi.Input<string> } = {
            APP_NAME: this.name,
            CLOUDCC_NAMESPACE: this.namespace,
            EXECUNIT_NAME: execUnitName,
            KLOTHO_S3_PREFIX: this.account.accountId,
        }

        const execEdgeID = this.topology.topologyIconData.find(
            (resource) => resource.kind == 'exec_unit' && resource.title == execUnitName
        )!.id

        if (env_vars) {
            for (const v of env_vars) {
                if (v.Kind == '') {
                    additionalEnvVars[v.Name] = v.Value
                } else {
                    const result = this.getEnvVarForDependency(v)
                    additionalEnvVars[result[0]] = result[1]
                }
            }
        }

        additionalEnvVars.SNS_ARN_BASE = pulumi.interpolate`arn:aws:sns:${this.region}:${this.account.accountId}`
        return additionalEnvVars
    }

    configureExecUnitPolicies() {
        const functionSet = new Set<string>()

        this.topology.topologyIconData.forEach((resource) => {
            if (resource.kind == Resource.gateway) {
                this.topology.topologyEdgeData.forEach((edge) => {
                    if (edge.source == resource.id) {
                        functionSet.add(this.resourceIdToResource.get(edge.target).title)
                    }
                })
            }

            if (resource.kind == Resource.exec_unit) {
                this.topology.topologyEdgeData.forEach((edge) => {
                    if (edge.source == resource.id) {
                        const targetResource = this.resourceIdToResource.get(edge.target)
                        if (targetResource && targetResource.kind == Resource.exec_unit) {
                            if (this.execUnitToFunctions.has(targetResource.title)) {
                                this.addPolicyStatementForName(resource.title, {
                                    Effect: 'Allow',
                                    Action: ['lambda:InvokeFunction'],
                                    Resource: [
                                        this.execUnitToFunctions.get(targetResource.title)!.arn,
                                    ],
                                })
                            } else if (['ecs', 'eks'].includes(targetResource.type)) {
                                this.addPolicyStatementForName(resource.title, {
                                    Effect: 'Allow',
                                    Action: ['servicediscovery:DiscoverInstances'],
                                    Resource: ['*'],
                                })
                            }
                        }
                    }
                })
            }
        })

        for (const [name, func] of this.execUnitToFunctions) {
            if (functionSet.has(name)) {
                this.addPolicyStatementForName(name, {
                    Effect: 'Allow',
                    Action: ['lambda:InvokeFunction'],
                    Resource: [func.invokeArn],
                })
            }
        }
    }

    createTopic(
        path: string,
        name: string,
        event: string,
        publishers: string[],
        subscribers: string[]
    ): aws.sns.Topic {
        let topic = `${this.name}_${name}_${event}`
        // validate rather than sanitize since the PubSub runtime depends on a specific topic format
        validate(topic, AwsSanitizer.SNS.topic.nameValidation())
        if (topic.length > 256) {
            const hash = crypto.createHash('sha256')
            hash.update(topic)
            topic = `${hash.digest('hex')}_${event}`
        }
        let sns = this.snsTopics.get(topic)
        if (!sns) {
            sns = new aws.sns.Topic(topic, {
                name: topic,
            })
            this.snsTopics.set(topic, sns)
        }

        for (const pub of publishers) {
            this.addPolicyStatementForName(pub, {
                Effect: 'Allow',
                Action: ['sns:Publish'],
                Resource: [sns.arn],
            })
        }

        for (const sub of subscribers) {
            const func = this.execUnitToFunctions.get(sub)!
            new aws.sns.TopicSubscription(
                `${topic}: ${sub}-subscription`,
                {
                    topic: sns.arn,
                    protocol: 'lambda',
                    endpoint: func.arn,
                },
                { parent: sns }
            )

            new aws.lambda.Permission(
                `${topic}-${sub}`,
                {
                    action: 'lambda:InvokeFunction',
                    function: func.arn,
                    principal: 'sns.amazonaws.com',
                    sourceArn: sns.arn,
                },
                { parent: func }
            )
        }

        return sns
    }

    setupKV(): aws.dynamodb.Table {
        const tableName = sanitized(AwsSanitizer.DynamoDB.table.nameValidation())`${h(this.name)}`
        const db = new aws.dynamodb.Table(
            `KV_${tableName}`,
            {
                attributes: [
                    { name: 'pk', type: 'S' },
                    { name: 'sk', type: 'S' },
                ],
                hashKey: 'pk',
                rangeKey: 'sk',
                billingMode: 'PAY_PER_REQUEST',
                ttl: {
                    // 'expiration' will only be set on items if TTL is enabled via annotation.
                    // At IaC-level, blanket enable and it will be ignored if not present on the item(s).
                    attributeName: 'expiration',
                    enabled: true,
                },
                name: tableName,
            },
            { protect: this.protect }
        )

        this.topology.topologyIconData.forEach((resource) => {
            if (resource.kind == Resource.kv) {
                this.topology.topologyEdgeData.forEach((edge) => {
                    if (resource.id == edge.target) {
                        const name = this.resourceIdToResource.get(edge.source).title
                        this.addPolicyStatementForName(name, {
                            Effect: 'Allow',
                            Action: ['dynamodb:*'],
                            Resource: [
                                db.arn,
                                pulumi.interpolate`${db.arn}/index/*`,
                                pulumi.interpolate`${db.arn}/stream/*`,
                                pulumi.interpolate`${db.arn}/backup/*`,
                                pulumi.interpolate`${db.arn}/export/*`,
                            ],
                        })
                    }
                })
            }
        })
        return db
    }

    createBuckets(bucketsToCreate, forceDestroy = false) {
        bucketsToCreate.forEach((b) => {
            const bucketName = this.account.accountId.apply(
                (accountId) =>
                    sanitized(
                        AwsSanitizer.S3.bucket.nameValidation()
                    )`${accountId}-${this.name}-${this.region}-${b.Name}`
            )
            const bucket = new aws.s3.Bucket(
                b.Name,
                {
                    bucket: bucketName,
                    forceDestroy,
                },
                { protect: this.protect }
            )
            this.buckets.set(b.Name, bucket)

            this.topology.topologyIconData.forEach((resource) => {
                if (resource.kind == Resource.fs) {
                    this.topology.topologyEdgeData.forEach((edge) => {
                        if (resource.id == edge.target) {
                            const name = this.resourceIdToResource.get(edge.source).title
                            this.addPolicyStatementForName(name, {
                                Effect: 'Allow',
                                Action: ['s3:*'],
                                Resource: [bucket.arn, pulumi.interpolate`${bucket.arn}/*`],
                            })
                        }
                    })
                }
            })

            const nameSet = new Set<string>()
            this.topology.topologyEdgeData.forEach((edge) => {
                const source = this.resourceIdToResource.get(edge.source)
                const target = this.resourceIdToResource.get(edge.target)
                if (source.kind == Resource.exec_unit && target.kind == Resource.exec_unit) {
                    nameSet.add(source.title)
                    nameSet.add(target.title)
                } else if (source.kind == Resource.exec_unit && target.kind == Resource.pubsub) {
                    // pubsub publisher
                    nameSet.add(source.title)
                } else if (source.kind == Resource.pubsub && target.kind == Resource.exec_unit) {
                    // pubsub subscriber
                    nameSet.add(target.title)
                }
            })
            nameSet.forEach((n) => {
                this.addPolicyStatementForName(n, {
                    Effect: 'Allow',
                    Action: ['s3:*'],
                    Resource: [bucket.arn, pulumi.interpolate`${bucket.arn}/*`],
                })
            })
        })
    }

    private createExecutionRole(execUnitPhysicalName: string) {
        const roleName = sanitized(AwsSanitizer.IAM.role.nameValidation())`${h(
            this.name
        )}_${this.generateHashFromPhysicalName(execUnitPhysicalName)}_LambdaExec`
        const lambdaExecRole = new aws.iam.Role(roleName, {
            assumeRolePolicy: {
                Version: '2012-10-17',
                Statement: [
                    {
                        Action: 'sts:AssumeRole',
                        Principal: {
                            Service: 'lambda.amazonaws.com',
                        },
                        Effect: 'Allow',
                        Sid: '',
                    },
                    {
                        Action: 'sts:AssumeRole',
                        Principal: {
                            Service: 'ecs-tasks.amazonaws.com',
                        },
                        Effect: 'Allow',
                    },
                    {
                        Action: 'sts:AssumeRole',
                        Principal: {
                            Service: 'tasks.apprunner.amazonaws.com',
                        },
                        Effect: 'Allow',
                    },
                ],
            },
        })
        // https://docs.aws.amazon.com/lambda/latest/dg/monitoring-cloudwatchlogs.html#monitoring-cloudwatchlogs-prereqs
        new aws.iam.RolePolicyAttachment(`${this.name}-${execUnitPhysicalName}-lambdabasic`, {
            role: lambdaExecRole,
            policyArn: aws.iam.ManagedPolicies.AWSLambdaVPCAccessExecutionRole,
        })

        return lambdaExecRole
    }

    public installLambdaWarmer(execUnitNames) {
        let region = this.region
        const name = 'lambdaWarmer'
        const lambdaNames = execUnitNames.map((n) => `${this.name}-${n}`)

        const warmerRole = this.createRoleForName(name)
        const warmerFuncName = sanitized(AwsSanitizer.Lambda.lambdaFunction.nameValidation())`${h(
            this.name
        )}-lambdawarmer`
        let warmerLambda = new aws.lambda.CallbackFunction(name, {
            name: warmerFuncName,
            memorySize: 128 /*MB*/,
            timeout: 60,
            runtime: 'nodejs14.x',
            callback: async (e) => {
                function getRandomInt(max) {
                    return Math.floor(Math.random() * max)
                }

                for (const lambdaFuncName of lambdaNames) {
                    let invokeParams = {
                        FunctionName: lambdaFuncName,
                        InvocationType: 'Event',
                        Payload: JSON.stringify(['warmed up', getRandomInt(100) + '']),
                    }
                    let awsSdk = require('aws-sdk')
                    const lambda = new awsSdk.Lambda({ region: region })
                    await lambda.invoke(invokeParams).promise()
                }
            },

            role: warmerRole,
        })

        const functionArns: pulumi.Output<string>[] = []
        execUnitNames.forEach((n) => {
            if (this.execUnitToFunctions.has(n)) {
                const fn = this.execUnitToFunctions.get(n)!
                functionArns.push(fn.arn)
            }
        })

        this.addPolicyStatementForName(name, {
            Effect: 'Allow',
            Action: ['lambda:*'],
            Resource: [...functionArns],
        })

        const cloudwatchLogs = new aws.cloudwatch.LogGroup(`${name}-function-api-lg`, {
            name: pulumi.interpolate`/aws/lambda/${warmerLambda.id}`,
            retentionInDays: 1,
        })

        const warmUpLambda: aws.cloudwatch.EventRuleEventHandler = warmerLambda
        const warmUpLambdaSchedule: aws.cloudwatch.EventRuleEventSubscription =
            aws.cloudwatch.onSchedule('warmUpLambda', 'cron(0/5 * * * ? *)', warmUpLambda)
    }

    public scheduleFunction(execUnitName, moduleName, functionName, cronExpression) {
        const execGroupName = `${this.name}/${execUnitName}`
        const key = sha256.sync(cronExpression).slice(0, 5)
        const name = `${execGroupName}.${functionName}:${key}`
        const scheduleRole = this.createRoleForName(name)

        const schedulerFuncName = sanitized(
            AwsSanitizer.Lambda.lambdaFunction.nameValidation()
        )`${h(this.name)}/${h(execUnitName)}_${h(functionName)}-${key}`

        let lambdaScheduler = new aws.lambda.CallbackFunction(name, {
            name: schedulerFuncName,
            memorySize: 128 /*MB*/,
            timeout: 300,
            runtime: 'nodejs14.x',
            callback: async (e) => {
                console.log(
                    `Running scheduled call for ${execGroupName}_${moduleName}_${functionName} ${cronExpression}`
                )

                const payloadToSend = {
                    __moduleName: moduleName,
                    __functionToCall: functionName,
                    __params: '',
                    __callType: 'rpc',
                }

                let invokeParams = {
                    FunctionName: execGroupName,
                    InvocationType: 'Event',
                    Payload: JSON.stringify(payloadToSend),
                }
                let awsSdk = require('aws-sdk')
                const lambda = new awsSdk.Lambda({ region: this.region })
                await lambda.invoke(invokeParams).promise()
            },

            role: scheduleRole,
        })

        if (this.execUnitToFunctions.has(execGroupName)) {
            this.addPolicyStatementForName(name, {
                Effect: 'Allow',
                Action: ['lambda:*'],
                Resource: [this.execUnitToFunctions.get(execGroupName)!.arn],
            })
        }
        let cloudwatchLogs = new aws.cloudwatch.LogGroup(`${name}-function-api-lg`, {
            name: lambdaScheduler.id.apply(
                (id) =>
                    sanitized(AwsSanitizer.CloudWatch.logGroup.nameValidation())`/aws/lambda/${h(
                        id
                    )}`
            ),
            retentionInDays: 1,
        })

        const schedulerLambda: aws.cloudwatch.EventRuleEventHandler = lambdaScheduler
        const warmUpLambdaSchedule: aws.cloudwatch.EventRuleEventSubscription =
            aws.cloudwatch.onSchedule(
                sanitized(AwsSanitizer.EventBridge.rule.nameValidation())`${h(execGroupName)}_${h(
                    functionName
                )}_act`,
                `cron(${cronExpression})`,
                schedulerLambda
            )
    }

    public addConnectionString(orm: string, connectionString: pulumi.Output<string>) {
        const clients: string[] = []
        this.topology.topologyIconData.forEach((resource) => {
            if (resource.kind == Resource.orm) {
                this.topology.topologyEdgeData.forEach((edge) => {
                    var id = resource.id

                    if (edge.target == id && id == `${orm}_${Resource.orm}`) {
                        // stores our connection string in environment variables + print it to the console
                        this.connectionString.set(id, connectionString)
                        // store another copy of the connection string for helm environment variables
                        const envVar = `${resource.title}${resource.kind}`
                            .toUpperCase()
                            .replace(/[^a-z0-9]/gi, '')
                        this.connectionString.set(`${envVar}CONNECTION`, connectionString)

                        clients.push(edge.source)
                    }
                })
            }
        })
        return clients
    }

    public async uploadTopologyDiagramToPortal(deployId) {
        pulumi.all(this.topologySpecOutputs).apply(async (t) => {
            // image file generated by the visualization service
            const diagramFileName = `${pulumi.getStack()}.png`
            // json structure for the portal to build the interactive diagram
            const diagramStructureFileName = `${pulumi.getStack()}.json`

            const diagram = fs.readFileSync(diagramFileName, {
                encoding: 'base64',
            })
            const diagramJson = fs.readFileSync(diagramStructureFileName)

            // Upload the diagram image - TODO: remove later
            let data1 = {
                data: diagram,
            }

            const resp1 = await requestRetry({
                url: `https://app.klo.dev/v1/topology/diagram/${deployId}`,
                method: 'POST',
                json: true,
                body: data1,
                maxAttempts: 3,
                retryDelay: 100,
            })

            if (resp1.statusCode == 201) {
                console.log(
                    `Successfully uploaded ${diagramStructureFileName} for deploy: ${deployId}`
                )
            } else {
                console.error(`Failed to upload ${diagramStructureFileName}`)
            }

            // Upload the diagram json structure
            let data = {
                data: diagramJson,
            }

            const resp = await requestRetry({
                url: `https://app.klo.dev/v1/topology/diagram/${deployId}-v2`,
                method: 'POST',
                json: true,
                body: data,
                maxAttempts: 3,
                retryDelay: 100,
            })

            if (resp.statusCode == 201) {
                console.log(`Successfully uploaded ${diagramFileName} for deploy: ${deployId}`)
            } else {
                console.error(`Failed to upload ${diagramFileName}`)
            }
        })
    }

    public async uploadTopologySpecToPortal(deployId) {
        let spec: ResourceInfo[] = []

        pulumi.all(this.topologySpecOutputs).apply(async (t) => {
            t.forEach((r) => {
                spec.push({ ...r })
            })
            const data = {
                data: spec,
            }
            const resp = await requestRetry({
                url: `https://app.klo.dev/v1/topology/spec/${deployId}`,
                method: 'POST',
                json: true,
                body: data,
                maxAttempts: 3,
                retryDelay: 100,
            })

            if (resp.statusCode == 201) {
                console.log(`Successfully uploaded spec for deploy: ${deployId}`)
            } else {
                console.error(`Failed to upload spec for deploy: ${deployId}`)
            }
        })
    }

    public async uploadAnalytics(message: string, sendOnDryRun: boolean, waitOnInfra: boolean) {
        if (pulumi.runtime.isDryRun() != sendOnDryRun) {
            return
        }

        const user = await analytics.retrieveUser()

        if (waitOnInfra) {
            pulumi.all(this.topologySpecOutputs).apply(async (t) => {
                await analytics.sendAnalytics(user, message, this.name)
            })
        } else {
            await analytics.sendAnalytics(user, message, this.name)
        }
    }

    createRoleForName(name: string): aws.iam.Role {
        const roleName = sanitized(AwsSanitizer.IAM.role.nameValidation())`${name}`
        const role: aws.iam.Role = this.createExecutionRole(roleName)
        this.execUnitToRole.set(roleName, role)
        return role
    }

    generateHashFromPhysicalName(execUnitName: string): string {
        const nameHash: string = sha256.sync(execUnitName)
        return nameHash.slice(0, 5)
    }

    createEcsCluster() {
        const providedClustername = kloConfig.get<string>('cluster')

        if (providedClustername != undefined) {
            // Since we use awsx clusters, we cannot just use the cluster retrieved from this get cluster call.
            // instead, this serves as a way to validate a cluster with the provided name actually exists.
            // Pulumi will error if a cluster with the provided name doesn't exist while the awsx's cluster create will
            // simply ignore the provided name in that scenario. This get cluster serves as a way to prevent us from
            // creating a new cluster despite the user providing one.
            const cluster = aws.ecs.getCluster({ clusterName: providedClustername })
        }

        // set up service discovery
        this.privateDnsNamespace = new aws.servicediscovery.PrivateDnsNamespace(
            `${this.name}-privateDns`,
            {
                name: sanitized(
                    AwsSanitizer.ServiceDiscovery.privateDnsNamespace.nameValidation()
                )`${h(this.name)}-privateDns`,
                description: 'Used for service discovery',
                vpc: this.klothoVPC.id,
            }
        )

        this.cluster = new awsx.ecs.Cluster(
            sanitized(AwsSanitizer.ECS.cluster.nameValidation())`${h(this.name)}-cluster`,
            {
                vpc: this.klothoVPC,
                cluster: providedClustername,
                securityGroups: [], // otherwise, awsx creates a default one with 0.0.0.0/0. See #314
            }
        )
    }

    createEksResources = async (
        execUnits: EksExecUnit[],
        charts?: HelmChart[],
        lbPlugin?: LoadBalancerPlugin
    ) => {
        let clusterName = sanitized(AwsSanitizer.EKS.cluster.nameValidation())`${h(
            this.name
        )}-eks-cluster`
        const providedClustername = kloConfig.get<string>('eks-cluster')
        const existingCluster = undefined
        if (this.eks == undefined) {
            if (providedClustername != undefined) {
                const existingCluster: aws.eks.GetClusterResult = await aws.eks.getCluster({
                    name: providedClustername!,
                })
                clusterName = providedClustername
            }
        }
        this.eks = new Eks(
            clusterName,
            DefaultEksClusterOptions,
            this,
            execUnits,
            charts || [],
            existingCluster,
            lbPlugin
        )
    }

    createEcsService(
        execUnitName,
        baseArgs: Partial<awsx.ecs.Container>,
        network_placement: 'public' | 'private',
        envVars: any,
        lbPlugin?: LoadBalancerPlugin
    ) {
        if (this.cluster == undefined) {
            this.createEcsCluster()
        }

        const image = this.execUnitToImage.get(execUnitName)!

        const role = this.createRoleForName(execUnitName)

        this.addPolicyStatementForName(execUnitName, {
            Effect: 'Allow',
            Action: ['ssm:GetParameters', 'secretsmanager:GetSecretValue', 'kms:Decrypt', 'ecr:*'],
            Resource: '*',
        })
        let needsLoadBalancer = false
        this.topology.topologyIconData.forEach((resource) => {
            if (resource.kind === Resource.gateway) {
                this.topology.topologyEdgeData.forEach((edge) => {
                    if (edge.source == resource.id && edge.target === `${execUnitName}_exec_unit`) {
                        // We know that this exec unit is exposed and must create the necessary resources
                        if (resource.type == 'apigateway') {
                            needsLoadBalancer = true
                        }
                    }
                })
            }
        })

        let nlb
        if (needsLoadBalancer) {
            nlb = new awsx.lb.NetworkLoadBalancer(
                sanitized(AwsSanitizer.ELB.loadBalancer.nameValidation())`${h(execUnitName)}-nlb`,
                {
                    external: false,
                    vpc: this.klothoVPC,
                    subnets: this.privateSubnetIds,
                }
            )
            this.execUnitToNlb.set(execUnitName, nlb)

            const targetGroup: awsx.elasticloadbalancingv2.NetworkTargetGroup =
                nlb.createTargetGroup(
                    sanitized(AwsSanitizer.ELB.targetGroup.nameValidation())`${h(
                        execUnitName
                    )})-tg`,
                    {
                        port: 3000,
                    }
                )

            const listener = targetGroup.createListener(`${execUnitName}-listener`, {
                port: 80,
            })

            const vpcLink = new aws.apigateway.VpcLink(`${execUnitName}-vpc-link`, {
                targetArn: nlb.loadBalancer.arn,
            })

            this.execUnitToVpcLink.set(execUnitName, vpcLink)
        }

        const logGroupName = sanitized(
            AwsSanitizer.CloudWatch.logGroup.nameValidation()
        )`/aws/fargate/${h(this.name)}-${h(execUnitName)}-task`

        let cloudwatchGroup = new aws.cloudwatch.LogGroup(`${this.name}-${execUnitName}-lg`, {
            name: `${logGroupName}`,
            retentionInDays: 1,
        })

        let additionalEnvVars: { name: string; value: pulumi.Input<string> }[] = []
        for (const [name, value] of Object.entries(
            this.generateExecUnitEnvVars(execUnitName, envVars)
        )) {
            additionalEnvVars.push({ name, value })
        }

        const serviceName = sanitized(AwsSanitizer.ServiceDiscovery.service.nameValidation())`${h(
            execUnitName
        )}`
        const discoveryService = new aws.servicediscovery.Service(serviceName, {
            name: serviceName,
            dnsConfig: {
                namespaceId: this.privateDnsNamespace.id,
                dnsRecords: [
                    {
                        ttl: 10,
                        type: 'A',
                    },
                ],
                routingPolicy: 'MULTIVALUE',
            },
            healthCheckCustomConfig: {
                failureThreshold: 1,
            },
        })

        const task = new awsx.ecs.FargateTaskDefinition(`${execUnitName}-task`, {
            logGroup: cloudwatchGroup,
            family: sanitized(AwsSanitizer.ECS.taskDefinition.familyValidation())`${h(
                execUnitName
            )}-family`,
            executionRole: role,
            taskRole: role,
            container: {
                ...baseArgs,
                image: image,
                portMappings:
                    nlb != undefined
                        ? nlb.listeners
                        : [
                              {
                                  containerPort: 3000,
                                  protocol: 'tcp',
                              },
                          ],
                environment: [
                    { name: 'AWS_XRAY_CONTEXT_MISSING', value: 'LOG_ERROR' },
                    ...additionalEnvVars,
                ],
                logConfiguration: {
                    logDriver: 'awslogs',
                    options: {
                        'awslogs-group': `${logGroupName}`,
                        'awslogs-region': `${this.region}`,
                        'awslogs-stream-prefix': '/ecs',
                    },
                },
            },
        })

        // This is done here for now because of a potential deletion race condition mentioned on the pulumi site
        const attach = new aws.iam.RolePolicyAttachment(`${execUnitName}-fargateAttach`, {
            role: role.name,
            policyArn: 'arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy',
        })

        const subnets =
            network_placement === 'public' ? this.publicSubnetIds : this.privateSubnetIds

        const service = new awsx.ecs.FargateService(
            `${execUnitName}-service`,
            {
                name: sanitized(AwsSanitizer.ECS.service.nameValidation())`${h(
                    execUnitName
                )}-service}`,
                cluster: this.cluster,
                taskDefinition: task,
                desiredCount: 1,
                subnets,
                securityGroups: this.sgs,
                serviceRegistries: {
                    registryArn: discoveryService.arn,
                },
            },
            {
                dependsOn: [attach],
            }
        )
    }

    public setupRedis = async (
        name: string,
        type: 'elasticache' | 'memorydb',
        args: Partial<aws.elasticache.ClusterArgs | aws.memorydb.ClusterArgs>
    ) => {
        if (type === 'elasticache') {
            const subnetGroup = new aws.elasticache.SubnetGroup(
                sanitized(
                    AwsSanitizer.Elasticache.cacheSubnetGroup.cacheSubnetGroupNameValidation()
                )`${h(this.name)}-${h(name)}-subnetgroup`,
                {
                    subnetIds: this.privateSubnetIds,
                    tags: {
                        Name: 'Klotho DB subnet group',
                    },
                }
            )
            args = args as aws.elasticache.ClusterArgs
            setupElasticacheCluster(
                name,
                args,
                this.topology,
                this.protect,
                this.connectionString,
                subnetGroup.name,
                this.sgs,
                this.name
            )
        } else if (type === 'memorydb') {
            // Since not all zones are supported in us-east-1 and us-west-2 we will verify our subnets are valid for the subnet group
            const supported_azs = [
                'use1-az2',
                'use1-az4',
                'use1-az6',
                'usw2-az1',
                'usw2-az2',
                'usw2-az3',
            ]
            let subnets: string[] | Promise<pulumi.Output<string>[]> = []
            if (['us-east-1', 'us-west-2'].includes(this.region)) {
                for (const subnetId in this.privateSubnetIds) {
                    const subnet: aws.ec2.GetSubnetResult = await aws.ec2.getSubnet({
                        id: subnetId,
                    })
                    if (supported_azs.includes(subnet.availabilityZoneId)) {
                        subnets.push(subnetId)
                    }
                }
                if (subnets.length === 0) {
                    throw new Error(
                        'Unable to find subnets in supported memorydb Availability Zones'
                    )
                }
            } else {
                subnets = this.privateSubnetIds
            }

            const subnetGroup = new aws.memorydb.SubnetGroup(
                sanitized(AwsSanitizer.MemoryDB.subnetGroup.subnetGroupNameValidation())`${
                    this.name
                }-${h(name)}-subnetgroup`,
                {
                    subnetIds: subnets,
                    tags: {
                        Name: 'Klotho DB subnet group',
                    },
                }
            )
            args = args as aws.memorydb.ClusterArgs
            setupMemoryDbCluster(
                name,
                args,
                this.topology,
                this.protect,
                this.connectionString,
                subnetGroup.name,
                this.sgs,
                this.name
            )
        }
    }
}
