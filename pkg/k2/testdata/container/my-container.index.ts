import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import * as command from '@pulumi/command'
import * as docker from '@pulumi/docker'
import * as pulumi from '@pulumi/pulumi'
import { OutputInstance } from '@pulumi/pulumi'


const kloConfig = new pulumi.Config('klo')
const protect = kloConfig.getBoolean('protect') ?? false
const awsConfig = new pulumi.Config('aws')
const awsProfile = awsConfig.get('profile')
const accountId = pulumi.output(aws.getCallerIdentity({}))
const region = pulumi.output(aws.getRegion({}))

const my_container_image_ecr_repo = new aws.ecr.Repository("my-container-image-ecr_repo", {
        imageScanningConfiguration: {
            scanOnPush: true,
        },
        imageTagMutability: 'MUTABLE',
        forceDelete: true,
        encryptionConfigurations: [{ encryptionType: 'KMS' }],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-image-ecr_repo"},
    })
const ecs_cluster_0 = new aws.ecs.Cluster("ecs_cluster-0", {
        settings: [{name: "containerInsights", value: "enabled"}],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "ecs_cluster-0"},
    })
const my_container_task_execution_role = new aws.iam.Role("my-container-task-execution-role", {
        assumeRolePolicy: pulumi.jsonStringify({Statement: [{Action: ["sts:AssumeRole"], Effect: "Allow", Principal: {Service: ["ecs-tasks.amazonaws.com"]}}], Version: "2012-10-17"}),
        managedPolicyArns: [
            ...["arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"],
        ],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-task-execution-role"},
    })
const my_container_task_log_group = new aws.cloudwatch.LogGroup("my-container-task-log-group", {
        name: "/aws/ecs/my-container-task",
        retentionInDays: 5,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-task-log-group"},
    })
const region_0 = pulumi.output(aws.getRegion({}))
const default_network_vpc = aws.ec2.Vpc.get("default-network-vpc", "FAKE (dryrun)")
const my_container_image = (() => {
        const base = new docker.Image(`${"my-container-image"}-base`, {
            build: {
                context: "/home/gordon/w/k2/pkg/k2/language_host/python/samples",
                dockerfile: "/home/gordon/w/k2/pkg/k2/language_host/python/samples/Dockerfile",
                platform: "linux/amd64",
            },
            skipPush: true,
            imageName: pulumi.interpolate`${my_container_image_ecr_repo.repositoryUrl}:base`,
        })

        const sha256 = base.repoDigest.apply((digest) => {
            return digest.substring(digest.indexOf('sha256:') + 7)
        })

        return new docker.Image(
            "my-container-image",
            {
                build: {
                    context: "/home/gordon/w/k2/pkg/k2/language_host/python/samples",
                    dockerfile: "/home/gordon/w/k2/pkg/k2/language_host/python/samples/Dockerfile",
                    platform: "linux/amd64",
                    cacheFrom: {
                        images: [base.imageName],
                    },
                },
                registry: aws.ecr
                    .getAuthorizationTokenOutput(
                        { registryId: my_container_image_ecr_repo.registryId },
                        { async: true }
                    )
                    .apply((registryToken) => {
                        return {
                            server: my_container_image_ecr_repo.repositoryUrl,
                            username: registryToken.userName,
                            password: registryToken.password,
                        }
                    }),
                imageName: pulumi.interpolate`${my_container_image_ecr_repo.repositoryUrl}:${sha256}`,
            },
            { parent: base }
        )
    })()
const my_container_lb_security_group = new aws.ec2.SecurityGroup("my-container-lb-security_group", {
        name: "my-container-lb-security_group",
        vpcId: default_network_vpc.id,
        egress: [{cidrBlocks: ["0.0.0.0/0"], description: "Allows all outbound IPv4 traffic", fromPort: 0, protocol: "-1", toPort: 0}],
        ingress: [{cidrBlocks: ["0.0.0.0/0"], description: "Allow ingress traffic from within the same security group", fromPort: 0, protocol: "-1", toPort: 0}, {description: "Allow ingress traffic from within the same security group", fromPort: 0, protocol: "-1", self: true, toPort: 0}],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-lb-security_group"},
    })
const my_container_service_security_group = new aws.ec2.SecurityGroup("my-container-service-security_group", {
        name: "my-container-service-security_group",
        vpcId: default_network_vpc.id,
        egress: [{cidrBlocks: ["0.0.0.0/0"], description: "Allows all outbound IPv4 traffic", fromPort: 0, protocol: "-1", toPort: 0}],
        ingress: [{cidrBlocks: ["10.0.0.0/18"], description: "Allow ingress traffic from ip addresses within the subnet default-network-public-subnet-1", fromPort: 0, protocol: "-1", toPort: 0}, {cidrBlocks: ["10.0.64.0/18"], description: "Allow ingress traffic from ip addresses within the subnet default-network-public-subnet-2", fromPort: 0, protocol: "-1", toPort: 0}, {description: "Allow ingress traffic from within the same security group", fromPort: 0, protocol: "-1", self: true, toPort: 0}],
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-service-security_group"},
    })
const default_network_private_subnet_1 = aws.ec2.Subnet.get("default-network-private-subnet-1", "FAKE (dryrun)")
const default_network_private_subnet_2 = aws.ec2.Subnet.get("default-network-private-subnet-2", "FAKE (dryrun)")
const my_container_tg = (() => {
        const tg = new aws.lb.TargetGroup("my-container-tg", {
            port: 80,
            protocol: "TCP",
            targetType: "ip",
            vpcId: default_network_vpc.id,
            healthCheck: {
    enabled: true,
    healthyThreshold: 5,
    interval: 30,
    protocol: "TCP",
    timeout: 5,
    unhealthyThreshold: 2
},
            tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-tg"},
        })
        return tg
    })()
const my_container_task = new aws.ecs.TaskDefinition("my-container-task", {
        family: "my-container-task",
        cpu: "256",
        memory: "512",
        networkMode: "awsvpc",
        requiresCompatibilities: ["FARGATE"],
        executionRoleArn: my_container_task_execution_role.arn,
        taskRoleArn: my_container_task_execution_role.arn,
        containerDefinitions: pulumi.jsonStringify([
    {
        cpu: 256,
        essential: true,
        image: my_container_image.imageName,
        logConfiguration: {
            logDriver: "awslogs",
            options: {
                "awslogs-group": "/aws/ecs/my-container-task",
                "awslogs-region": region_0.apply((o) => o.name),
                "awslogs-stream-prefix": "my-container-task-my-container-task",
            },
        },
        memory: 512,
        name: "my-container-task",
        portMappings: [
            {
                containerPort: 80,
                hostPort: 80,
                protocol: "TCP",
            },
        ],        
    },
]),
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-task"},
    })
const default_network_public_subnet_1 = aws.ec2.Subnet.get("default-network-public-subnet-1", "FAKE (dryrun)")
const default_network_public_subnet_2 = aws.ec2.Subnet.get("default-network-public-subnet-2", "FAKE (dryrun)")
const my_container_service = new aws.ecs.Service(
        "my-container-service",
        {
            launchType: "FARGATE",
            cluster: ecs_cluster_0.arn,
            desiredCount: 1,
            forceNewDeployment: true,
            loadBalancers: [
    {
        containerPort: 80,
        targetGroupArn: my_container_tg.arn,
        containerName: "my-container-task",
    },
]

,
            networkConfiguration: {
                subnets: [default_network_private_subnet_1, default_network_private_subnet_2].map((sn) => sn.id),
                securityGroups: [my_container_service_security_group].map((sg) => sg.id),
            },
            taskDefinition: my_container_task.arn,
            waitForSteadyState: true,
            //TMP
            tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-service"},
        },
        { dependsOn: [default_network_private_subnet_1, default_network_private_subnet_2, ecs_cluster_0, my_container_service_security_group, my_container_task, my_container_tg] }
    )
const my_container_lb = new aws.lb.LoadBalancer("my-container-lb", {
        internal: false,
        loadBalancerType: "network",
        subnets: [default_network_public_subnet_1, default_network_public_subnet_2].map((subnet) => subnet.id),
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-lb"},
        securityGroups: [my_container_lb_security_group].map((sg) => sg.id),
    })
export const my_container_lb_DomainName = my_container_lb.dnsName
const my_container_service_cpuutilization = new aws.cloudwatch.MetricAlarm("my-container-service-CPUUtilization", {
        comparisonOperator: "GreaterThanOrEqualToThreshold",
        evaluationPeriods: 2,
        actionsEnabled: true,
        alarmDescription: "This metric checks for CPUUtilization in the ECS service",
        dimensions: {ClusterName: ecs_cluster_0.name, ServiceName: my_container_service.name},
        metricName: "CPUUtilization",
        namespace: "AWS/ECS",
        period: 60,
        statistic: "Average",
        threshold: 90,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-service-CPUUtilization"},
    })
const my_container_service_memoryutilization = new aws.cloudwatch.MetricAlarm("my-container-service-MemoryUtilization", {
        comparisonOperator: "GreaterThanOrEqualToThreshold",
        evaluationPeriods: 2,
        actionsEnabled: true,
        alarmDescription: "This metric checks for MemoryUtilization in the ECS service",
        dimensions: {ClusterName: ecs_cluster_0.name, ServiceName: my_container_service.name},
        metricName: "MemoryUtilization",
        namespace: "AWS/ECS",
        period: 60,
        statistic: "Average",
        threshold: 90,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-service-MemoryUtilization"},
    })
const my_container_service_runningtaskcount = new aws.cloudwatch.MetricAlarm("my-container-service-RunningTaskCount", {
        comparisonOperator: "LessThanThreshold",
        evaluationPeriods: 1,
        actionsEnabled: true,
        alarmDescription: "This metric checks for any stopped tasks in the ECS service",
        dimensions: {ClusterName: ecs_cluster_0.name, ServiceName: my_container_service.name},
        metricName: "RunningTaskCount",
        namespace: "ECS/ContainerInsights",
        period: 60,
        statistic: "Average",
        threshold: 1,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-service-RunningTaskCount"},
    })
const my_container_listener = new aws.lb.Listener("my-container-listener", {
        loadBalancerArn: my_container_lb.arn,
        defaultActions: [
    {
        targetGroupArn: my_container_tg.arn,
        type: "forward",
    },
]

,
        port: 80,
        protocol: "TCP",
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "my-container-listener"},
    })
const cloudwatch_dashboard_0 = new aws.cloudwatch.Dashboard("cloudwatch_dashboard-0", {
        dashboardName: "cloudwatch_dashboard-0",
        dashboardBody: pulumi.jsonStringify({widgets: [{height: 6, properties: {annotations: {alarms: [my_container_service_cpuutilization.arn]}, region: region_0.apply((o) => o.name)}, type: "metric", width: 6}, {height: 6, properties: {alarms: [my_container_service_cpuutilization.arn]}, type: "alarm", width: 6}, {height: 6, properties: {annotations: {alarms: [my_container_service_memoryutilization.arn]}, region: region_0.apply((o) => o.name)}, type: "metric", width: 6}, {height: 6, properties: {alarms: [my_container_service_memoryutilization.arn]}, type: "alarm", width: 6}, {height: 6, properties: {annotations: {alarms: [my_container_service_runningtaskcount.arn]}, region: region_0.apply((o) => o.name)}, type: "metric", width: 6}, {height: 6, properties: {alarms: [my_container_service_runningtaskcount.arn]}, type: "alarm", width: 6}]}),
    })

export const $outputs = {
	LoadBalancerUrl: pulumi.interpolate`http://${my_container_lb.dnsName}`,
}

export const $urns = {
	"aws:security_group:default-network-vpc:my-container-service-security_group": (my_container_service_security_group as any).urn,
	"aws:subnet:default-network-vpc:default-network-private-subnet-2": (default_network_private_subnet_2 as any).urn,
	"aws:ecs_task_definition:my-container-task": (my_container_task as any).urn,
	"aws:ecs_service:my-container-service": (my_container_service as any).urn,
	"aws:region:region-0": (region_0 as any).urn,
	"aws:ecr_image:my-container-image": (my_container_image as any).urn,
	"aws:target_group:my-container-tg": (my_container_tg as any).urn,
	"aws:subnet:default-network-vpc:default-network-public-subnet-2": (default_network_public_subnet_2 as any).urn,
	"aws:cloudwatch_alarm:my-container-service-CPUUtilization": (my_container_service_cpuutilization as any).urn,
	"aws:cloudwatch_alarm:my-container-service-MemoryUtilization": (my_container_service_memoryutilization as any).urn,
	"aws:ecs_cluster:ecs_cluster-0": (ecs_cluster_0 as any).urn,
	"aws:iam_role:my-container-task-execution-role": (my_container_task_execution_role as any).urn,
	"aws:vpc:default-network-vpc": (default_network_vpc as any).urn,
	"aws:load_balancer:my-container-lb": (my_container_lb as any).urn,
	"aws:security_group:default-network-vpc:my-container-lb-security_group": (my_container_lb_security_group as any).urn,
	"aws:subnet:default-network-vpc:default-network-private-subnet-1": (default_network_private_subnet_1 as any).urn,
	"aws:subnet:default-network-vpc:default-network-public-subnet-1": (default_network_public_subnet_1 as any).urn,
	"aws:cloudwatch_alarm:my-container-service-RunningTaskCount": (my_container_service_runningtaskcount as any).urn,
	"aws:load_balancer_listener:my-container-lb:my-container-listener": (my_container_listener as any).urn,
	"aws:cloudwatch_dashboard:cloudwatch_dashboard-0": (cloudwatch_dashboard_0 as any).urn,
	"aws:ecr_repo:my-container-image-ecr_repo": (my_container_image_ecr_repo as any).urn,
	"aws:log_group:my-container-task-log-group": (my_container_task_log_group as any).urn,
}
