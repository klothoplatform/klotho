import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
    Region: pulumi.Output<pulumi.UnwrappedObject<aws.GetRegionResult>>
    ServiceName: string
    VpcEndpointType: string
    Subnets: aws.ec2.Subnet[]
    SecurityGroupIds: pulumi.Input<string[]> | undefined
    RouteTables: aws.ec2.RouteTable[]
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.VpcEndpoint {
    return new aws.ec2.VpcEndpoint(args.Name, {
        vpcId: args.Vpc.id,
        serviceName: pulumi.interpolate`com.amazonaws.${args.Region.name}.${args.ServiceName}`,
        vpcEndpointType: args.VpcEndpointType,
        //TMPL {{- if eq .VpcEndpointType "Interface"}}
        privateDnsEnabled: true,
        subnetIds: args.Subnets.map((x) => x.id),
        securityGroupIds: args.SecurityGroupIds,
        //TMPL {{- end }}
        //TMPL {{- if and .RouteTables (eq .VpcEndpointType "Gateway")}}
        routeTableIds: args.RouteTables.map((rt) => rt.id),
        //TMPL {{- end}}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
