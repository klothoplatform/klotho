import * as aws from '@pulumi/aws'

interface Args {
    Name: string
}

function create(args: Args): aws.acm.Certificate {
    return new aws.acm.Certificate(args.Name, {})
}
