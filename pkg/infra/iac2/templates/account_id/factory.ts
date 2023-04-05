import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi.Output<pulumi.UnwrappedObject<aws.GetCallerIdentityResult>> {
    return pulumi.output(aws.getCallerIdentity({}))
}
