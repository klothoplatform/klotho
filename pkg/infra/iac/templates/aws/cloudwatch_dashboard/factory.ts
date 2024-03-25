import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    DashboardBody: Record<string, any>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.cloudwatch.Dashboard {
    return new aws.cloudwatch.Dashboard(args.Name, {
        dashboardName: args.Name,
        dashboardBody: pulumi.jsonStringify(args.DashboardBody),
    })
}

function properties(object: aws.cloudwatch.Dashboard, args: Args) {
    return {
        Arn: object.dashboardArn,
    }
}
