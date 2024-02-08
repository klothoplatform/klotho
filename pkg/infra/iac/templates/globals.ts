import * as pulumi from '@pulumi/pulumi'
import * as aws from '@pulumi/aws'

export const kloConfig = new pulumi.Config('klo')
export const protect = kloConfig.getBoolean('protect') ?? false
export const awsConfig = new pulumi.Config('aws')
export const awsProfile = awsConfig.get('profile')

export const accountId = pulumi.output(aws.getCallerIdentity({}))
export const region = pulumi.output(aws.getRegion({}))
