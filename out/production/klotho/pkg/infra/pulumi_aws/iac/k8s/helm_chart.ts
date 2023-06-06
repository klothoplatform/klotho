import * as pulumi_k8s from '@pulumi/kubernetes'
import * as pulumi from '@pulumi/pulumi'
import { CloudCCLib } from '../../deploylib'
import { Eks } from '../eks'
import { LoadBalancerPlugin } from '../load_balancing'

export class Value {
    public ExecUnitName: string
    public Kind: string
    public Type: string
    public Key: string
    public EnvironmentVariable: any
}

enum ValueTypes {
    TargetGroupTransformation = 'target_group',
    ImageTransformation = 'image',
    EnvironmentVariableTransformation = 'env_var',
    ServiceAccountAnnotationTransformation = 'service_account_annotation',
}

export const getChartValues = (
    lib: CloudCCLib,
    eks: Eks,
    transformations: Value[],
    lbPlugin?: LoadBalancerPlugin
): {
    [x: string]: any
} => {
    const values = {}
    transformations.forEach((t: Value) => {
        switch (t.Type) {
            case ValueTypes.ImageTransformation:
                values[t.Key] = lib.execUnitToImage.get(t.ExecUnitName)!
                break
            case ValueTypes.ServiceAccountAnnotationTransformation:
                values[t.Key] = lib.execUnitToRole.get(t.ExecUnitName)!.arn
                break
            case ValueTypes.TargetGroupTransformation:
                const targetGroup = lbPlugin!.execUnitToTargetGroup.get(t.ExecUnitName)!
                values[t.Key] = targetGroup.arn
                break
            case ValueTypes.EnvironmentVariableTransformation:
                // Currently the only env vars we set are persist related
                // This will need to be changed to be more extensible
                values[t.Key] = lib.getEnvVarForDependency(t.EnvironmentVariable)[1]
                break
            default:
                throw new Error(`Unsupported Transformation Type ${t.Key}`)
        }
    })
    return values
}

interface applyChartParams {
    eks: Eks
    lbPlugin?: LoadBalancerPlugin
    chartName: string
    values: Value[]
    dependsOn: any[]
    provider: pulumi_k8s.Provider
}

export const applyChart = (lib: CloudCCLib, args: applyChartParams) => {
    const values = getChartValues(lib, args.eks, args.values, args.lbPlugin)

    const transformation = (obj, opts): void => {
        if (obj.kind == 'TargetGroupBinding') {
            const execUnitName = obj.metadata.name
            const targetGroup = args.lbPlugin!.execUnitToTargetGroup.get(execUnitName)!
            obj.metadata.name = pulumi.interpolate`${execUnitName}-${targetGroup.arn.apply((arn) =>
                arn.substring(arn.length - 7)
            )}`
        }
        return
    }

    new pulumi_k8s.helm.v3.Chart(
        `${args.eks.clusterName}-${args.chartName}`,
        {
            path: `./charts/${args.chartName}`,
            values,
            transformations: [transformation],
        },
        { dependsOn: args.dependsOn, provider: args.provider }
    )
}
