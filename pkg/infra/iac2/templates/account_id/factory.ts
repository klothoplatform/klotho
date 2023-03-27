import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {}

// noinspection JSUnusedLocalSymbols
async function create(args: Args): Promise<aws.GetCallerIdentityResult> {
    return await aws.getCallerIdentity({})
}
