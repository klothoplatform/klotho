import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi.Output<pulumi.UnwrappedObject<aws.GetRegionResult>> {
    return pulumi.output(aws.getRegion({}))
}
