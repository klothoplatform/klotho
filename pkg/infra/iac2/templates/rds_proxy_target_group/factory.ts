import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    RdsInstance: aws.rds.Instance
    RdsProxy: aws.rds.Proxy
    ConnectionPoolConfigurationInfo: aws_native.types.input.rds.DBProxyTargetGroupConnectionPoolConfigurationInfoFormatArgs
    TargetGroupName: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.rds.ProxyTarget {
    return new aws.rds.ProxyTarget('exampleProxyTarget', {
        dbInstanceIdentifier: args.RdsInstance.id,
        dbProxyName: args.RdsProxy.name,
        targetGroupName: args.TargetGroupName,
    })
}
