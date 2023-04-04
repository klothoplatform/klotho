import * as pulumi from '@pulumi/pulumi'
import * as docker from '@pulumi/docker'
import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Repo: aws.ecr.Repository
    Context: string
    Dockerfile: string
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

function create(args: Args): docker.Image {
    return new docker.Image(args.Name, {
        build: {
            context: args.Context,
            dockerfile: args.Dockerfile,
            platform: 'linux/amd64',
        },
        imageName: pulumi.interpolate`${args.Repo.repositoryUrl}/${args.Name}`,
    })
}
