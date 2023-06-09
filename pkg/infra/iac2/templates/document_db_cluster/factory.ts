import * as aws from '@pulumi/aws'
import * as pulumi from "@pulumi/pulumi";

interface Args {
    Name: string
    AvailabilityZones: pulumi.Output<pulumi.UnwrappedObject<aws.GetAvailabilityZonesArgs>>
    MasterPassword: string
    MasterUsername: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.docdb.Cluster {
    return new aws.docdb.Cluster(args.Name, {
        availabilityZones: args.AvailabilityZones.apply(async (azArgs) =>
           (await aws.getAvailabilityZones(azArgs)).zoneIds.sort()
        ),
        masterPassword: args.MasterPassword,
        masterUsername: args.MasterUsername,
    })
}
