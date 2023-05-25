import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    protect: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.secretsmanager.Secret {
    return new aws.secretsmanager.Secret(
        args.Name,
        {
            name: args.Name,
            recoveryWindowInDays: 0,
        },
        { protect: args.protect }
    )
}
