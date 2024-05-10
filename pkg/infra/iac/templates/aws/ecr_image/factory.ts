import * as pulumi from '@pulumi/pulumi'
import * as docker from '@pulumi/docker'
import * as aws from '@pulumi/aws'
import * as command from '@pulumi/command'

interface Args {
    Name: string
    Tag: string
    Repo: aws.ecr.Repository
    Context: string
    Dockerfile: string
    BaseImage: string
    Platform: string
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): docker.Image {
    return (() => {
        const base = new docker.Image(`${args.Name}-base`, {
            build: {
                context: args.Context,
                dockerfile: args.Dockerfile,
                platform: args.Platform,
            },
            skipPush: true,
            imageName: pulumi.interpolate`${args.Repo.repositoryUrl}:{{ if .Tag }}${args.Tag}-{{ end }}base`,
        })

        const sha256 = base.repoDigest.apply((digest) => {
            return digest.substring(digest.indexOf('sha256:') + 7)
        })

        return new docker.Image(
            args.Name,
            {
                build: {
                    context: args.Context,
                    dockerfile: args.Dockerfile,
                    platform: args.Platform,
                    cacheFrom: {
                        images: [base.imageName],
                    },
                },
                registry: aws.ecr
                    .getAuthorizationTokenOutput(
                        { registryId: args.Repo.registryId },
                        { async: true }
                    )
                    .apply((registryToken) => {
                        return {
                            server: args.Repo.repositoryUrl,
                            username: registryToken.userName,
                            password: registryToken.password,
                        }
                    }),
                imageName: pulumi.interpolate`${args.Repo.repositoryUrl}:{{ if .Tag }}${args.Tag}-{{ end }}${sha256}`,
            },
            { parent: base }
        )
    })()
}

function properties(object: docker.Image, args: Args) {
    return {
        ImageName: object.imageName,
    }
}
