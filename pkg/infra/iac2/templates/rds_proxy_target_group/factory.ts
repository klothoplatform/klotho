import * as aws_native from '@pulumi/aws-native'
import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    RdsInstance: aws.rds.Instance
    RdsProxy: aws.rds.Proxy
    ConnectionPoolConfigurationInfo: aws_native.types.input.rds.DBProxyTargetGroupConnectionPoolConfigurationInfoFormatArgs
    TargetGroupName: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws_native.rds.DBProxyTargetGroup {
    return new aws_native.rds.DBProxyTargetGroup(args.Name, {
        dBInstanceIdentifiers: [args.RdsInstance.identifier],
        dBProxyName: args.RdsProxy.name,
        connectionPoolConfigurationInfo: args.ConnectionPoolConfigurationInfo,
        targetGroupName: 'default',
    })
}
