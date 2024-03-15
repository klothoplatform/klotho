import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    CidrBlock: string
    Vpc: aws.ec2.Vpc
    AvailabilityZone: pulumi.Output<string>
    MapPublicIpOnLaunch: boolean
    Id?: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.Subnet {
    return new aws.ec2.Subnet(args.Name, {
        vpcId: args.Vpc.id,
        cidrBlock: args.CidrBlock,
        availabilityZone: args.AvailabilityZone,
        mapPublicIpOnLaunch: args.MapPublicIpOnLaunch,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ec2.Subnet, args: Args) {
    return {
        Id: object.id,
    }
}


function importResource(args: Args): aws.ec2.Subnet {
    return aws.ec2.Subnet.get(args.Name, args.Id)
}
