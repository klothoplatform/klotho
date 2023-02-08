import * as aws from '@pulumi/aws'
import * as awsx from '@pulumi/awsx'
import * as eks from '@pulumi/eks'
import * as pulumi from '@pulumi/pulumi'
import { ClusterArgs, FargateProfileArgs } from '@pulumi/aws/eks'
import * as pulumi_k8s from '@pulumi/kubernetes'
import * as k8s from './kubernetes'
import * as https from 'https'
import { getIssuerCAThumbprint } from '@pulumi/eks/cert-thumprint'
import { cloud_map_controller, alb_controller, external_dns } from './k8s/add_ons/'
import { local } from '@pulumi/command'
import { CloudCCLib, Resource } from '../deploylib'
import * as uuid from 'uuid'
import {
    createPodAutoScalerResourceMetric,
    autoScaleDeployment,
} from './k8s/horizontal-pod-autoscaling'
import { installMetricsServer } from './k8s/add_ons/metrics_server'
import { applyChart, Value } from './k8s/helm_chart'
import { LoadBalancerPlugin } from './load_balancing'

export interface EksExecUnitArgs {
    nodeType: 'fargate' | 'node'
    replicas: number
    nodeConstraints?: {
        diskSize?: number
        instanceType?: string
    }
    limits?: {
        cpu?: number
        memory?: number // Differs from ephemeral-storage resource request (stable in k8s 1.25)
    }
    autoscalingConfig?: {
        cpuUtilization?: number
        memoryUtilization?: number // Differs from ephemeral-storage resource request (stable in k8s 1.25)
        maxReplicas?: number
    }
    stickinessTimeout?: number
}

export interface HelmOptions {
    directory?: string
    install?: boolean
    values_file?: string
}

export interface EksExecUnit {
    name: string
    network_placement?: string
    params: EksExecUnitArgs
    helmOptions?: HelmOptions
    envVars?: any
}

export interface HelmChart {
    Name: string
    Directory: string
    Values: Value[] | null
}

interface FargateProfileSelector {
    namespace: string
    labels?: { [key: string]: string }
}

// ExecEnvVar sets the environment variables using an exec-based auth plugin.
interface ExecEnvVar {
    /**
     * Name of the auth exec environment variable.
     */
    name: pulumi.Input<string>

    /**
     * Value of the auth exec environment variable.
     */
    value: pulumi.Input<string>
}

/**
 * KubeconfigOptions represents the AWS credentials to scope a given kubeconfig
 * when using a non-default credential chain.
 *
 * The options can be used independently, or additively.
 *
 * A scoped kubeconfig is necessary for certain auth scenarios. For example:
 *   1. Assume a role on the default account caller,
 *   2. Use an AWS creds profile instead of the default account caller,
 *   3. Use an AWS creds creds profile instead of the default account caller,
 *      and then assume a given role on the profile. This scenario is also
 *      possible by only using a profile, iff the profile includes a role to
 *      assume in its settings.
 *
 * See for more details:
 * - https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html
 * - https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-role.html
 * - https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
 */
export interface KubeconfigOptions {
    /**
     * Role ARN to assume instead of the default AWS credential provider chain.
     *
     * The role is passed to kubeconfig as an authentication exec argument.
     */
    roleArn?: pulumi.Input<aws.ARN>
    /**
     * AWS credential profile name to always use instead of the
     * default AWS credential provider chain.
     *
     * The profile is passed to kubeconfig as an authentication environment
     * setting.
     */
    profileName?: pulumi.Input<string>
}

export enum plugins {
    AWS_LOAD_BALANCER_CONTROLLER,
    CERT_MANAGER,
    CLOUD_MAP_CONTROLLER,
    VPC_CNI,
    EXTERNAL_DNS,
    METRICS_SERVER,
}

interface EksClusterOptions {
    initializePluginsOnFargate?: boolean
    installPlugins?: plugins[]
    enableFargateLogging?: Boolean
    adminRoleArn?: string
    autoApply?: boolean
    createNodeGroup?: boolean
}

export const DefaultEksClusterOptions: EksClusterOptions = {
    initializePluginsOnFargate: true,
    installPlugins: [
        plugins.VPC_CNI,
        plugins.METRICS_SERVER,
        plugins.CERT_MANAGER,
        plugins.AWS_LOAD_BALANCER_CONTROLLER,
        plugins.CLOUD_MAP_CONTROLLER,
    ],
    enableFargateLogging: true,
    autoApply: true,
    createNodeGroup: true,
}

interface NodeGroupSpecs {
    diskSize?: number
    instanceType?: string
}

const KUBE_SYSTEM_NAMESPACE = 'kube-system'
const CLOUD_MAP_NAMESPACE = 'cloud-map-mcs-system'
// This is temporary until we expand EKS functionality
export const EXEC_UNIT_NAMESPACE = 'default'

export class Eks {
    private oidcProvider: aws.iam.OpenIdConnectProvider

    private namespaces: Map<string, pulumi_k8s.core.v1.Namespace> = new Map<
        string,
        pulumi_k8s.core.v1.Namespace
    >()

    // Service Accounts are serviceAccountName_namespace -> IAM Role *https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html*
    public readonly serviceAccounts: Map<string, aws.iam.Role> = new Map<string, aws.iam.Role>()

    // We want to keep track of installed plugins to allow for dependencies between them (ex. alb controller needs cert-manager, etc)
    public readonly installedPlugins: Map<plugins, pulumi.Resource | pulumi_k8s.helm.v3.Chart> =
        new Map<plugins, pulumi.Resource>()

    // Service Accounts are serviceAccountName_namespace -> IAM Role *https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html*
    public readonly execUnitToTargetGroupArn: Map<string, pulumi.Output<string>> = new Map<
        string,
        pulumi.Output<string>
    >()

    private pluginFargateProfile: aws.eks.FargateProfile
    private execUnitFargateProfile: aws.eks.FargateProfile

    private privateNodeGroups: Map<string, aws.eks.NodeGroup> = new Map<string, aws.eks.NodeGroup>()
    private publicNodeGroups: Map<string, aws.eks.NodeGroup> = new Map<string, aws.eks.NodeGroup>()

    private clusterSet: pulumi_k8s.yaml.ConfigFile

    private cluster: aws.eks.Cluster
    private kubeconfig: pulumi.Output<string>
    private k8sAdminRole: aws.iam.Role
    private provider: pulumi_k8s.Provider
    private serverSideApplyProvider: pulumi_k8s.Provider
    private readonly lib: CloudCCLib

    public readonly clusterName: string
    public readonly region: string
    public readonly options: EksClusterOptions

    constructor(
        clusterName: string,
        options: EksClusterOptions = DefaultEksClusterOptions,
        lib: CloudCCLib,
        execUnits: EksExecUnit[],
        charts: HelmChart[],
        existingCluster?: aws.eks.GetClusterResult,
        private readonly lbPlugin?: LoadBalancerPlugin
    ) {
        this.lib = lib
        this.options = options
        const config = new pulumi.Config('aws')
        this.region = config.get('region')!
        const vpc = lib.klothoVPC
        const securityGroupIds = lib.sgs

        this.clusterName = clusterName

        let clusterSubnetIds: pulumi.Input<pulumi.Input<string>[]> = lib.privateSubnetIds
        if (execUnits.filter((unit) => unit.network_placement === 'public').length > 0) {
            clusterSubnetIds = pulumi
                .all([lib.publicSubnetIds || [], lib.privateSubnetIds || []])
                .apply(([publicIds, privateIds]) => {
                    return [...publicIds, ...privateIds]
                })
        }

        if (!existingCluster) {
            this.k8sAdminRole = createAdminRole(clusterName)
            const args: ClusterArgs = {
                vpcConfig: {
                    subnetIds: clusterSubnetIds,
                    securityGroupIds,
                },
                roleArn: this.k8sAdminRole.arn,
            }
            this.cluster = new aws.eks.Cluster(clusterName, args)
        } else {
            this.cluster = aws.eks.Cluster.get(clusterName, existingCluster.id)
        }

        pulumi.output(vpc.privateSubnets).apply((ps) => {
            const cidrBlocks: any = ps.map((subnet) => subnet.subnet.cidrBlock)
            new aws.ec2.SecurityGroupRule(`${clusterName}-ingress`, {
                type: 'ingress',
                cidrBlocks: cidrBlocks,
                fromPort: 0,
                protocol: '-1',
                toPort: 0,
                securityGroupId: this.cluster.vpcConfig.clusterSecurityGroupId,
            })
        })

        this.kubeconfig = this.generateKubeconfig().apply(JSON.stringify)
        this.provider = new pulumi_k8s.Provider(`${clusterName}-eks-provider`, {
            kubeconfig: this.kubeconfig,
        })
        this.serverSideApplyProvider = new pulumi_k8s.Provider(
            `${clusterName}-eks-server-provider`,
            {
                kubeconfig: this.kubeconfig,
                enableServerSideApply: true,
            }
        )

        this.oidcProvider = this.setupOidc()

        if (options.initializePluginsOnFargate) {
            const podExecutionRole = this.createPodExecutionRole(`${clusterName}-plugin`)
            const selectors = [
                {
                    namespace: KUBE_SYSTEM_NAMESPACE,
                },
            ]
            this.pluginFargateProfile = this.createFargateProfile(
                `${clusterName}-plugin-profile`,
                podExecutionRole,
                vpc,
                selectors
            )
            this.patchCoreDNSDeployment()
        }

        this.createNodeGroups(execUnits)

        if (options.enableFargateLogging) {
            this.enableFargateLogging()
        }

        const p = options.installPlugins ? options.installPlugins : []
        for (const plugin of p) {
            let dependsOn: (pulumi.Resource | pulumi.Output<pulumi.CustomResource[]>)[] =
                options.initializePluginsOnFargate ? [this.pluginFargateProfile] : []
            const certManagerInstall: pulumi_k8s.helm.v3.Release = this.installedPlugins.get(
                plugins.CERT_MANAGER
            ) as pulumi_k8s.helm.v3.Release
            switch (plugin) {
                case plugins.VPC_CNI:
                    const vpcCni = new eks.VpcCni(`${clusterName}-vpc-cni-plugin`, this.kubeconfig)
                    this.installedPlugins.set(plugins.VPC_CNI, vpcCni)
                    break
                case plugins.METRICS_SERVER:
                    const metricsServer = installMetricsServer(clusterName, this.provider)
                    this.installedPlugins.set(plugins.METRICS_SERVER, metricsServer)
                    break
                case plugins.CERT_MANAGER:
                    const certManager = new pulumi_k8s.helm.v3.Release(
                        `${clusterName}-cert-manager`,
                        {
                            name: 'cert-manager',
                            chart: 'cert-manager',
                            repositoryOpts: { repo: 'https://charts.jetstack.io' },
                            values: {
                                installCRDs: true,
                                webhook: {
                                    timeoutSeconds: 30,
                                },
                            },
                            namespace: EXEC_UNIT_NAMESPACE,
                            version: 'v1.10.0',
                        },
                        { provider: this.provider, deleteBeforeReplace: true }
                    )
                    this.installedPlugins.set(plugins.CERT_MANAGER, certManager)
                    break
                case plugins.AWS_LOAD_BALANCER_CONTROLLER:
                    const lbSaName = `${clusterName}-alb-controller`
                    const lbServiceAccount = this.createServiceAccount(
                        lbSaName,
                        KUBE_SYSTEM_NAMESPACE
                    )
                    alb_controller.attachPermissionsToRole(
                        this.serviceAccounts.get(`${lbSaName}_${KUBE_SYSTEM_NAMESPACE}`)!
                    )
                    certManagerInstall ? dependsOn.push(certManagerInstall) : null
                    const albController = alb_controller.installLoadBalancerController(
                        clusterName,
                        KUBE_SYSTEM_NAMESPACE,
                        lbServiceAccount,
                        vpc,
                        this.provider,
                        this.region,
                        this.options.initializePluginsOnFargate || false,
                        dependsOn
                    )
                    this.installedPlugins.set(plugins.AWS_LOAD_BALANCER_CONTROLLER, albController)
                    break
                case plugins.CLOUD_MAP_CONTROLLER:
                    const chart: pulumi_k8s.helm.v3.Chart = this.installedPlugins.get(
                        plugins.AWS_LOAD_BALANCER_CONTROLLER
                    ) as pulumi_k8s.helm.v3.Chart
                    certManagerInstall ? dependsOn.push(certManagerInstall) : null
                    const cloudMapController = cloud_map_controller.installCloudMapController(
                        clusterName,
                        this.provider,
                        dependsOn
                    )
                    this.installedPlugins.set(plugins.CLOUD_MAP_CONTROLLER, cloudMapController)
                    const cloudMapSaName = 'cloud-map-mcs-controller-manager'
                    this.patchServiceAccount(cloudMapSaName, CLOUD_MAP_NAMESPACE, [
                        cloudMapController,
                    ])
                    const cloudMapRole: aws.iam.Role = this.serviceAccounts.get(
                        `${cloudMapSaName}_${CLOUD_MAP_NAMESPACE}`
                    )!
                    new aws.iam.RolePolicyAttachment('cloudmap-policy-attachment', {
                        policyArn: 'arn:aws:iam::aws:policy/AWSCloudMapFullAccess',
                        role: cloudMapRole,
                    })
                    this.clusterSet = cloud_map_controller.createClusterSet(
                        clusterName,
                        this.provider,
                        [cloudMapController, chart.ready, ...dependsOn]
                    )
                    break
                case plugins.EXTERNAL_DNS:
                    const dnsSaName = `${clusterName}-external-dns`
                    const dnsServiceAccount = this.createServiceAccount(
                        dnsSaName,
                        KUBE_SYSTEM_NAMESPACE
                    )
                    external_dns.attachPermissionsToRole(
                        this.serviceAccounts.get(`${dnsSaName}_${KUBE_SYSTEM_NAMESPACE}`)!
                    )
                    const externalDns = external_dns.installExternalDNS(
                        KUBE_SYSTEM_NAMESPACE,
                        this.provider,
                        dnsServiceAccount,
                        dependsOn
                    )
                    this.installedPlugins.set(plugins.EXTERNAL_DNS, externalDns)
                    break
            }
        }

        const podExecutionRole = this.createPodExecutionRole(`${clusterName}-default`)
        const selectors = [
            {
                namespace: 'default',
                labels: {
                    'klotho-fargate-enabled': 'true',
                },
            },
        ]
        this.execUnitFargateProfile = this.createFargateProfile(
            `${clusterName}-execunit-profile`,
            podExecutionRole,
            vpc,
            selectors
        )

        for (const unit of execUnits) {
            this.setupExecUnit(lib, unit)
        }
        for (const chart of charts) {
            this.setupKlothoHelmChart(lib, chart.Name, chart.Values || [])
        }
    }

    private determineNodeGroupSpecs(execUnits: EksExecUnit[]): Map<string, NodeGroupSpecs> {
        let nodeGroupSpecs: Map<string, NodeGroupSpecs> = new Map<string, NodeGroupSpecs>()
        const defaultDiskSize = 20
        const defaultInstanceType = 't3.medium'
        let diskSizeMin = defaultDiskSize

        if (execUnits.length == 0) {
            return nodeGroupSpecs
        }

        for (const unit of execUnits) {
            if (
                unit.params.nodeType === 'fargate' ||
                unit.params.nodeConstraints?.instanceType !== undefined
            ) {
                continue
            }
            if (unit.params.nodeConstraints?.diskSize !== undefined) {
                diskSizeMin =
                    diskSizeMin < unit.params.nodeConstraints.diskSize
                        ? unit.params.nodeConstraints.diskSize
                        : diskSizeMin
            }
        }

        for (const unit of execUnits) {
            if (unit.params.nodeType === 'fargate') {
                continue
            }
            if (unit.params.nodeConstraints) {
                const instanceType = unit.params.nodeConstraints.instanceType
                const diskSize = unit.params.nodeConstraints.diskSize

                if (instanceType) {
                    const key = instanceType
                    const currentDiskSize = nodeGroupSpecs.get(key)?.diskSize

                    if (!currentDiskSize && diskSize) {
                        nodeGroupSpecs.set(key, {
                            ...nodeGroupSpecs[key],
                            diskSize: diskSize > diskSizeMin ? diskSize : diskSizeMin,
                            instanceType,
                        })
                    } else if (currentDiskSize && diskSize) {
                        nodeGroupSpecs.set(key, {
                            ...nodeGroupSpecs[key],
                            diskSize: diskSize > currentDiskSize ? diskSize : currentDiskSize,
                            instanceType,
                        })
                    } else if (!currentDiskSize && !diskSize) {
                        nodeGroupSpecs.set(key, {
                            ...nodeGroupSpecs.get(key),
                            diskSize: diskSizeMin,
                            instanceType,
                        })
                    }
                }
            }
        }
        // if there are no node group specs, create our default node group as t3.medium
        nodeGroupSpecs.size === 0
            ? nodeGroupSpecs.set(defaultInstanceType, {
                  diskSize: diskSizeMin,
                  instanceType: defaultInstanceType,
              })
            : null

        return nodeGroupSpecs
    }

    private createNodeGroups(execUnits: EksExecUnit[]) {
        const privateNodeGroupSpecs = this.determineNodeGroupSpecs(
            execUnits.filter((unit) => unit.network_placement !== 'public')
        )
        const publicNodeGroupSpecs = this.determineNodeGroupSpecs(
            execUnits.filter((unit) => unit.network_placement === 'public')
        )
        if (privateNodeGroupSpecs.size == 0 && publicNodeGroupSpecs.size == 0) {
            privateNodeGroupSpecs.set('t3.medium', { diskSize: 20, instanceType: 't3.medium' })
        }

        privateNodeGroupSpecs.forEach((specs, name) => {
            const nodeGroup = this.createNodeGroup(
                `private-${name}`.replace('.', '-'),
                this.lib.privateSubnetIds,
                specs,
                { network_placement: 'private' }
            )
            this.privateNodeGroups.set(name, nodeGroup)
        })

        publicNodeGroupSpecs.forEach((specs, name) => {
            const nodeGroup = this.createNodeGroup(
                `public-${name}`.replace('.', '-'),
                this.lib.publicSubnetIds,
                specs,
                { network_placement: 'public' }
            )
            this.publicNodeGroups.set(name, nodeGroup)
        })
    }

    private createNodeGroup(
        nodeGroupName: string,
        subnetIds: Promise<pulumi.Output<string>[]> | string[],
        specs: NodeGroupSpecs,
        labels?: pulumi.Input<{
            [key: string]: pulumi.Input<string>
        }>
    ): aws.eks.NodeGroup {
        const nodeRole = this.createNodeIamRole(nodeGroupName)
        return new aws.eks.NodeGroup(`${nodeGroupName}-NodeGroup`, {
            clusterName: this.cluster.name,
            nodeRoleArn: nodeRole.arn,
            subnetIds,
            scalingConfig: {
                desiredSize: 2,
                maxSize: 2,
                minSize: 1,
            },
            updateConfig: {
                maxUnavailable: 1,
            },
            diskSize: specs.diskSize!,
            instanceTypes: [specs.instanceType!],
            labels,
        })
    }

    public setupKlothoHelmChart(lib: CloudCCLib, name: string, values: Value[]) {
        const chart: pulumi_k8s.helm.v3.Chart = this.installedPlugins.get(
            plugins.AWS_LOAD_BALANCER_CONTROLLER
        ) as pulumi_k8s.helm.v3.Chart
        if (chart == undefined) {
            throw new Error('Cannot create ServiceExport without Cloud Map Controller installed')
        }

        let sleep
        if (this.options.initializePluginsOnFargate) {
            sleep = new local.Command(
                'sleepForCerts',
                {
                    create: 'sleep 60',
                    update: 'sleep 60',
                    triggers: [uuid.v4()],
                },
                { dependsOn: chart.ready }
            )
        } else {
            sleep = new local.Command(
                'sleepForCerts',
                {
                    create: 'sleep 10',
                    update: 'sleep 10',
                    triggers: [uuid.v4()],
                },
                { dependsOn: chart.ready }
            )
        }

        applyChart(lib, {
            eks: this,
            lbPlugin: this.lbPlugin,
            chartName: name,
            values,
            dependsOn: [
                this.execUnitFargateProfile,
                chart.ready,
                this.installedPlugins.get(plugins.CLOUD_MAP_CONTROLLER)!,
                sleep,
            ],
            provider: this.provider,
        })
    }

    private setupExecUnit(lib: CloudCCLib, unit: EksExecUnit) {
        const execUnit = unit.name
        const image = this.lib.execUnitToImage.get(unit.name)!
        const args = unit.params
        let dependencyParent
        let serviceName
        let role
        const execUnitEdgeId = `${execUnit}_exec_unit`
        let needsProxy = false
        let needsGatewayLink = false
        let needsLoadBalancer = false

        lib.topology.topologyIconData.forEach((resource) => {
            if (resource.kind === Resource.gateway) {
                lib.topology.topologyEdgeData.forEach((edge) => {
                    if (edge.source == resource.id && edge.target === execUnitEdgeId) {
                        // We know that this exec unit is exposed and must create the necessary resources
                        needsGatewayLink = true
                        if (resource.type == 'apigateway') {
                            needsLoadBalancer = true
                        }
                    }
                })
            }
            if (resource.kind === Resource.exec_unit) {
                lib.topology.topologyEdgeData.forEach((edge) => {
                    if (edge.source == resource.id && edge.target === execUnitEdgeId) {
                        // We know that this exec unit is proxied to from another exec unit and must create the necessary resources
                        needsProxy = true
                    }
                })
            }
        })
        const protocol = needsLoadBalancer ? 'TCP' : 'HTTP'

        let additionalEnvVars: { name: string; value: pulumi.Input<string> }[] = []
        for (const [name, value] of Object.entries(
            lib.generateExecUnitEnvVars(execUnit, unit.envVars)
        )) {
            additionalEnvVars.push({ name, value })
        }
        let nodeSelector: { [key: string]: pulumi.Output<string> | string } = {}

        if (unit.params.nodeType !== 'fargate') {
            nodeSelector['network_placement'] = unit.network_placement!
            if (args.nodeConstraints?.instanceType) {
                if (unit.network_placement === 'public') {
                    nodeSelector['eks.amazonaws.com/nodegroup'] = this.publicNodeGroups.get(
                        `${args.nodeConstraints.instanceType}`
                    )!.nodeGroupName
                } else {
                    nodeSelector['eks.amazonaws.com/nodegroup'] = this.privateNodeGroups.get(
                        `${args.nodeConstraints.instanceType}`
                    )!.nodeGroupName
                }
            }
        }

        if (image) {
            role = this.createServiceAccountRole(execUnit, EXEC_UNIT_NAMESPACE)
            lib.execUnitToRole.set(execUnit, role)
        }

        if (!unit.helmOptions?.install) {
            const sa = this.createServiceAccount(execUnit, EXEC_UNIT_NAMESPACE, role)
            const deployment = this.createDeployment(
                execUnit,
                image,
                args,
                additionalEnvVars,
                sa,
                nodeSelector
            )
            if (args.autoscalingConfig) {
                const cpuUtil = args.autoscalingConfig.cpuUtilization
                const memUtil = args.autoscalingConfig.memoryUtilization
                const maxReplicas = args.autoscalingConfig.maxReplicas || args.replicas * 2
                const metrics: pulumi_k8s.types.input.autoscaling.v2.MetricSpec[] = []
                if (cpuUtil) {
                    metrics.push(createPodAutoScalerResourceMetric('cpu', 'Utilization', cpuUtil))
                }
                if (memUtil) {
                    metrics.push(
                        createPodAutoScalerResourceMetric('memory', 'Utilization', memUtil)
                    )
                }
                autoScaleDeployment({
                    deploymentName: execUnit,
                    minReplicas: args.replicas,
                    maxReplicas,
                    metrics,
                    provider: this.provider,
                    dependsOn: [deployment],
                })
            }
            const service = this.createService(execUnit, lib.klothoVPC, args, deployment)
            serviceName = service.metadata.name
            dependencyParent = service
        }

        if (image) {
            if (needsGatewayLink) {
                if (!this.lbPlugin) {
                    throw new Error(
                        'EKS Plugin needs the LoadBalancer Plugin to connect expose with execution units.'
                    )
                }
                if (needsLoadBalancer) {
                    const lb = this.lbPlugin!.createLoadBalancer(lib.name, execUnit, {
                        subnets: lib.privateSubnetIds,
                        loadBalancerType: 'network',
                    })
                    const targetGroup = this.lbPlugin!.createTargetGroup(lib.name, execUnit, {
                        port: 3000,
                        protocol,
                        vpcId: lib.klothoVPC.id,
                        targetType: 'ip',
                    })

                    this.lbPlugin!.createListener(lib.name, execUnit, {
                        port: 80,
                        protocol,
                        loadBalancerArn: lb.arn,
                        defaultActions: [
                            {
                                type: 'forward',
                                targetGroupArn: targetGroup.arn,
                            },
                        ],
                    })

                    const vpcLink = new aws.apigateway.VpcLink(`${execUnit}-vpc-link`, {
                        targetArn: lb.arn,
                    })
                    lib.execUnitToVpcLink.set(execUnit, vpcLink)
                } else {
                    this.lbPlugin!.createTargetGroup(lib.name, execUnit, {
                        port: 3000,
                        protocol,
                        vpcId: lib.klothoVPC.id,
                        targetType: 'ip',
                    })

                    pulumi.output(this.lib.klothoVPC.publicSubnets).apply((ps) => {
                        const cidrBlocks: any = ps.map((subnet) => subnet.subnet.cidrBlock)
                        new aws.ec2.SecurityGroupRule(
                            `${this.lib.name}-${this.clusterName}-public-ingress`,
                            {
                                type: 'ingress',
                                cidrBlocks: cidrBlocks,
                                fromPort: 0,
                                protocol: '-1',
                                toPort: 0,
                                securityGroupId: this.cluster.vpcConfig.clusterSecurityGroupId,
                            }
                        )
                    })
                }

                if (
                    this.installedPlugins.get(plugins.AWS_LOAD_BALANCER_CONTROLLER) &&
                    !unit.helmOptions?.install
                ) {
                    const targetGroup = this.lbPlugin!.execUnitToTargetGroup.get(execUnit)!
                    this.createTargetBinding(execUnit, serviceName, targetGroup.arn, [
                        targetGroup,
                        dependencyParent,
                    ])
                }
            }

            if (needsProxy && !unit.helmOptions?.install) {
                if (this.installedPlugins.get(plugins.CLOUD_MAP_CONTROLLER)) {
                    this.createServiceExport(execUnit, [dependencyParent])
                }
            }
        }
    }

    public getExecUnitRole = (execUnit: string): aws.iam.Role | undefined => {
        return this.serviceAccounts.get(`${execUnit}_${EXEC_UNIT_NAMESPACE}`)
    }

    private generateKubeconfig(): { [key: string]: any } {
        const config = new pulumi.Config('aws')
        const profile = config.get('profile')

        let args = [
            'eks',
            'get-token',
            '--cluster-name',
            this.cluster.name,
            '--region',
            this.region,
        ]
        if (profile) {
            args.push('--profile', profile)
        }
        return pulumi
            .all([args, this.cluster.endpoint, this.cluster.certificateAuthorities[0].data])
            .apply(([tokenArgs, clusterEndpoint, certData]) => {
                return {
                    apiVersion: 'v1',
                    clusters: [
                        {
                            cluster: {
                                server: clusterEndpoint,
                                'certificate-authority-data': certData,
                            },
                            name: 'kubernetes',
                        },
                    ],
                    contexts: [
                        {
                            context: {
                                cluster: 'kubernetes',
                                user: 'aws',
                            },
                            name: 'aws',
                        },
                    ],
                    'current-context': 'aws',
                    kind: 'Config',
                    users: [
                        {
                            name: 'aws',
                            user: {
                                exec: {
                                    apiVersion: 'client.authentication.k8s.io/v1beta1',
                                    command: 'aws',
                                    args: tokenArgs,
                                },
                            },
                        },
                    ],
                }
            })
    }

    private patchCoreDNSDeployment(): pulumi_k8s.apps.v1.DeploymentPatch {
        return new pulumi_k8s.apps.v1.DeploymentPatch(
            'coreDNSDeploymentPatch',
            {
                metadata: {
                    annotations: {
                        'pulumi.com/patchForce': 'true',
                    },
                    name: 'coredns',
                    namespace: KUBE_SYSTEM_NAMESPACE,
                },
                spec: {
                    template: {
                        metadata: {
                            annotations: {
                                'eks.amazonaws.com/compute-type': '',
                            },
                        },
                    },
                },
            },
            { provider: this.serverSideApplyProvider, dependsOn: this.pluginFargateProfile }
        )
    }

    private setupOidc() {
        // Retrieve the OIDC provider URL's intermediate root CA fingerprint.
        const eksOidcProviderUrl = pulumi.interpolate`https://oidc.eks.${this.region}.amazonaws.com`
        const agent = new https.Agent({
            // Cached sessions can result in the certificate not being
            // available since its already been "accepted." Disable caching.
            maxCachedSessions: 0,
        })
        const fingerprint = getIssuerCAThumbprint(eksOidcProviderUrl, agent)

        // Create the OIDC provider for the cluster.
        return new aws.iam.OpenIdConnectProvider(`oidcProvider`, {
            clientIdLists: ['sts.amazonaws.com'],
            url: this.cluster.identities[0].oidcs[0].issuer,
            thumbprintLists: [fingerprint],
        })
    }

    private createServiceAccountRole(name: string, namespace: string): aws.iam.Role {
        const saKey = `${name}_${namespace}`
        if (this.serviceAccounts.get(saKey)) {
            throw new Error('Service Account already exists: ' + saKey)
        }
        const assumeRolePolicyDoc = pulumi
            .all([this.oidcProvider.arn, this.oidcProvider.url])
            .apply(([oidc_arn, oidc_url]) => {
                return {
                    Version: '2012-10-17',
                    Statement: [
                        {
                            Effect: 'Allow',
                            Principal: {
                                Federated: oidc_arn,
                            },
                            Action: 'sts:AssumeRoleWithWebIdentity',
                            Condition: {
                                StringEquals: {
                                    [`${oidc_url}:sub`]: `system:serviceaccount:${namespace}:${name}`,
                                },
                            },
                        },
                    ],
                }
            })
        const role = new aws.iam.Role(`${name}-role`, {
            assumeRolePolicy: assumeRolePolicyDoc.apply(JSON.stringify),
        })
        this.serviceAccounts.set(saKey, role)
        return role
    }

    private patchServiceAccount(
        name: string,
        namespace: string,
        dependsOn?: pulumi.Resource[]
    ): pulumi_k8s.core.v1.ServiceAccount {
        const role = this.createServiceAccountRole(name, namespace)
        const sa = new pulumi_k8s.core.v1.ServiceAccountPatch(
            name,
            {
                metadata: {
                    name,
                    annotations: {
                        'pulumi.com/patchForce': 'true',
                        test: 'test',
                        'eks.amazonaws.com/role-arn': role.arn,
                    },
                },
            },
            { dependsOn, provider: this.serverSideApplyProvider }
        )
        return sa
    }

    private createServiceAccount(
        name: string,
        namespace: string,
        role?: aws.iam.Role,
        dependsOn?: pulumi.Resource[]
    ): pulumi_k8s.core.v1.ServiceAccount {
        if (!role) {
            role = this.createServiceAccountRole(name, namespace)
        }
        const sa = new pulumi_k8s.core.v1.ServiceAccount(
            name,
            {
                automountServiceAccountToken: true,
                metadata: {
                    name,
                    namespace,
                    annotations: {
                        'eks.amazonaws.com/role-arn': role.arn,
                    },
                },
            },
            { provider: this.provider, dependsOn }
        )
        return sa
    }

    private createNamespace(
        name: string,
        labels?: { [key: string]: any }
    ): pulumi_k8s.core.v1.Namespace {
        if (this.namespaces.get(name)) {
            throw new Error(`Namespace ${name} already exists`)
        }
        const namespace = new pulumi_k8s.core.v1.Namespace(
            `${name}-ns`,
            {
                metadata: {
                    name,
                    labels,
                },
            },
            { provider: this.provider }
        )
        this.namespaces.set(name, namespace)
        return namespace
    }

    // Fargate Functions
    private createFargateProfile(
        profileName: string,
        podExecutionRole: aws.iam.Role,
        vpc: awsx.ec2.Vpc,
        selectors: FargateProfileSelector[],
        dependsOn?
    ): aws.eks.FargateProfile {
        const args: FargateProfileArgs = {
            clusterName: this.cluster.name,
            podExecutionRoleArn: podExecutionRole.arn,
            selectors,
            subnetIds: pulumi.output(vpc.privateSubnetIds),
        }
        const profile: aws.eks.FargateProfile = new aws.eks.FargateProfile(profileName, args, {
            dependsOn,
            customTimeouts: { create: '30m', update: '30m', delete: '30m' },
        })
        return profile
    }

    private enableFargateLogging() {
        const ns = 'aws-observability'
        const labels = { 'aws-observability': 'enabled' }
        this.createNamespace(ns, labels)

        const configMap = new pulumi_k8s.core.v1.ConfigMap(
            'aws-observability-configmap',
            {
                metadata: {
                    name: 'aws-logging',
                    namespace: ns,
                },
                data: {
                    'output.conf': `[OUTPUT]
                    Name cloudwatch_logs
                    Match   *
                    region ${this.region}
                    log_group_name fluent-bit-cloudwatch
                    log_stream_prefix from-fluent-bit-
                    auto_create_group true
                    log_key log`,

                    'parsers.conf': `[PARSER]
                    Name crio
                    Format Regex
                    Regex ^(?<time>[^ ]+) (?<stream>stdout|stderr) (?<logtag>P|F) (?<log>.*)$
                    Time_Key    time
                    Time_Format %Y-%m-%dT%H:%M:%S.%L%z`,

                    'filters.conf': `[FILTER]
                    Name parser
                    Match *
                    Key_name log
                    Parser crio`,
                },
            },
            { provider: this.provider }
        )
    }

    // These are exec unit level methods

    public createDeployment(
        execUnit: string,
        image,
        args: EksExecUnitArgs,
        envVars: { name: string; value: pulumi.Input<string> }[],
        sa: pulumi_k8s.core.v1.ServiceAccount,
        nodeSelector?: { [key: string]: string | pulumi.Output<string> }
    ): pulumi_k8s.apps.v1.Deployment {
        const env = [
            ...envVars,
            {
                name: 'AWS_REGION',
                value: this.region,
            },
        ]

        const resources: pulumi_k8s.types.input.core.v1.ResourceRequirements = {
            limits: {},
            requests: {},
        }
        // Assume the request for the container is half of the limit for both memory and cpu
        if (args.limits?.cpu) {
            resources.requests!['cpu'] = `${args.limits?.cpu * 1000}m`
        }
        if (args.limits?.memory) {
            resources.limits!['memory'] = `${args.limits?.memory}Mi`
            resources.requests!['memory'] = `${args.limits?.memory}Mi`
        }

        return k8s.createDeployment({
            name: execUnit,
            image,
            replicas: args.replicas,
            appLabels: generateLabels(execUnit, args.nodeType === 'fargate'),
            env,
            serviceAccountName: sa.metadata.name,
            resources,
            nodeSelector,
            k8sProvider: this.provider,
            parent: this.execUnitFargateProfile,
        })
    }

    public createService(
        execUnit: string,
        vpc: awsx.ec2.Vpc,
        args: EksExecUnitArgs,
        parent,
        dependsOn: (pulumi.Resource | pulumi.Output<pulumi.CustomResource[]>)[] = [],
        serviceDiscoveryDomain?: pulumi.Output<string>,
        createNLB?: boolean
    ) {
        const metadataAnnotations = createNLB
            ? {
                  'service.beta.kubernetes.io/aws-load-balancer-type': 'external',
                  'service.beta.kubernetes.io/aws-load-balancer-nlb-target-type': 'ip',
                  'service.beta.kubernetes.io/aws-load-balancer-name': execUnit,
                  'service.beta.kubernetes.io/aws-load-balancer-subnets': pulumi.interpolate`${vpc.privateSubnetIds}`,
              }
            : {}
        if (serviceDiscoveryDomain != undefined) {
            metadataAnnotations[
                'external-dns.alpha.kubernetes.io/hostname'
            ] = `${execUnit}.${serviceDiscoveryDomain}`
            const chart: pulumi_k8s.helm.v3.Chart = this.installedPlugins.get(
                plugins.EXTERNAL_DNS
            ) as pulumi_k8s.helm.v3.Chart
            if (chart == undefined) {
                throw new Error(
                    'Cannot create TargetGroupBinding without AWS Load Balancer Controller installed'
                )
            }
            dependsOn.push(chart.ready)
        }
        const chart: pulumi_k8s.helm.v3.Chart = this.installedPlugins.get(
            plugins.AWS_LOAD_BALANCER_CONTROLLER
        ) as pulumi_k8s.helm.v3.Chart
        chart ? dependsOn.push(chart.ready) : null
        return k8s.createService(
            execUnit,
            this.provider,
            generateLabels(execUnit, args.nodeType === 'fargate'),
            metadataAnnotations,
            args.stickinessTimeout ? args.stickinessTimeout : 0,
            parent,
            dependsOn
        )
    }

    public createServiceExport(execUnit: string, dependsOn: pulumi.Resource[]) {
        const chart: pulumi_k8s.helm.v3.Chart = this.installedPlugins.get(
            plugins.AWS_LOAD_BALANCER_CONTROLLER
        ) as pulumi_k8s.helm.v3.Chart
        if (chart == undefined) {
            throw new Error('Cannot create ServiceExport without Cloud Map Controller installed')
        }
        cloud_map_controller.createServiceExport(
            execUnit,
            EXEC_UNIT_NAMESPACE,
            this.provider,
            null,
            [chart, this.clusterSet, ...dependsOn]
        )
    }

    public createTargetBinding(
        execUnit: string,
        serviceName: pulumi.Output<string>,
        targetGroupArn: pulumi.Output<string>,
        dependsOn: pulumi.Resource[]
    ) {
        const chart: pulumi_k8s.helm.v3.Chart = this.installedPlugins.get(
            plugins.AWS_LOAD_BALANCER_CONTROLLER
        ) as pulumi_k8s.helm.v3.Chart
        if (chart == undefined) {
            throw new Error(
                'Cannot create TargetGroupBinding without AWS Load Balancer Controller installed'
            )
        }
        const sleep = new local.Command(
            'sleepForCerts',
            {
                create: 'sleep 60',
                update: 'sleep 60',
                triggers: [uuid.v4()],
            },
            { dependsOn: chart.ready }
        )
        alb_controller.createTargetBinding(
            execUnit,
            serviceName,
            80,
            targetGroupArn,
            this.provider,
            null,
            [sleep, ...dependsOn]
        )
    }

    private createPodExecutionRole(roleName: string): aws.iam.Role {
        const podExecutionRole = new aws.iam.Role(`${roleName}-fargate`, {
            assumeRolePolicy: {
                Version: '2012-10-17',
                Statement: [
                    {
                        Action: 'sts:AssumeRole',
                        Principal: {
                            Service: 'eks-fargate-pods.amazonaws.com',
                        },
                        Effect: 'Allow',
                        Sid: '',
                    },
                ],
            },
            managedPolicyArns: ['arn:aws:iam::aws:policy/AmazonEKSFargatePodExecutionRolePolicy'],
        })

        new aws.iam.RolePolicy(`${roleName}-fargateloggingpolicy`, {
            role: podExecutionRole,
            policy: JSON.stringify({
                Version: '2012-10-17',
                Statement: [
                    {
                        Effect: 'Allow',
                        Action: [
                            'logs:CreateLogStream',
                            'logs:CreateLogGroup',
                            'logs:DescribeLogStreams',
                            'logs:PutLogEvents',
                        ],
                        Resource: '*',
                    },
                ],
            }),
        })
        return podExecutionRole
    }

    private createNodeIamRole(name: string): aws.iam.Role {
        let roleName = `${this.clusterName}-${name}-EksNodeRole`
        if (roleName.length > 55) {
            roleName = roleName.substring(0, 55)
        }
        return new aws.iam.Role(roleName, {
            assumeRolePolicy: {
                Version: '2012-10-17',
                Statement: [
                    {
                        Action: 'sts:AssumeRole',
                        Principal: {
                            Service: 'ec2.amazonaws.com',
                        },
                        Effect: 'Allow',
                        Sid: '',
                    },
                ],
            },
            managedPolicyArns: [
                'arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy',
                'arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly',
                'arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy',
                'arn:aws:iam::aws:policy/AWSCloudMapFullAccess',
            ],
        })
    }
}

export const createAdminRole = (clusterName: string): aws.iam.Role => {
    const k8sAdminRole = new aws.iam.Role(`${clusterName}_k8sAdmin`, {
        assumeRolePolicy: {
            Version: '2012-10-17',
            Statement: [
                {
                    Action: 'sts:AssumeRole',
                    Principal: {
                        Service: 'eks.amazonaws.com',
                    },
                    Effect: 'Allow',
                    Sid: '',
                },
            ],
        },
        managedPolicyArns: ['arn:aws:iam::aws:policy/AmazonEKSClusterPolicy'],
    })
    return k8sAdminRole
}

const generateLabels = (execUnit: string, fargateEnabled: boolean): { [key: string]: string } => {
    return { 'klotho-fargate-enabled': fargateEnabled.toString(), execUnit }
}
