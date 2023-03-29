import * as aws from '@pulumi/aws'

interface Args {}

// noinspection JSUnusedLocalSymbols
function create(args: Args): Promise<aws.GetRegionResult> {
    return aws.getRegion({})
}
