import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    Index: number
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi.Output<pulumi.UnwrappedObject<aws.GetAvailabilityZonesArgs>> {
    return pulumi.output(
        aws.getAvailabilityZones({
            state: 'available',
        })
    ).names[args.Index]
}

function importResource(
    args: Args
): pulumi.Output<pulumi.UnwrappedObject<aws.GetAvailabilityZonesArgs>> {
    return pulumi.output(
        aws.getAvailabilityZones({
            state: 'available',
        })
    ).names[args.Index]
}
