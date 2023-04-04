import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
    Region: pulumi.Output<Promise<aws.GetRegionResult>>
    ServiceName: string
    VpcEndpointType: string
    Subnets: aws.ec2.Subnet[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.VpcEndpoint {
    return new aws.ec2.VpcEndpoint(args.Name, {
        vpcId: args.Vpc.id,
        serviceName: pulumi.interpolate`com.amazonaws.${args.Region.name}.${args.ServiceName}`,
        vpcEndpointType: args.VpcEndpointType,
        //TMPL {{ if eq .VpcEndpointType.Raw "Interface"}}
        privateDnsEnabled: true,
        subnetIds: args.Subnets.map((x) => x.id),
        //TMPL {{ end }}
        //TMPL {{ if eq .VpcEndpointType.Raw "Gateway"}}
        routeTableIds: [args.Vpc.defaultRouteTableId.apply((id) => id)],
        //TMPL {{ end}}
    })
}
