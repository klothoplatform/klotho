import * as aws from '@pulumi/aws'

interface Args {
    Name: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecs.Cluster {
    return new aws.ecs.Cluster(args.Name, {})
}
