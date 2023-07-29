import * as pulumi from '@pulumi/pulumi'
import * as docker from '@pulumi/docker'
import * as aws from '@pulumi/aws'
import * as command from '@pulumi/command'

interface Args {
    Name: string
    TagBase: string
    Repo: aws.ecr.Repository
    Context: string
    Dockerfile: string
    BaseImage: string
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): docker.Image {
    return (() => {
        //TMPL {{- if .BaseImage.Raw }}
        const pullBaseImage = new command.local.Command(
            `${args.Name}-pull-base-image-${Date.now()}`,
            { create: pulumi.interpolate`docker pull ${args.BaseImage}` }
        )
        //TMPL {{- end }}
        const base = new docker.Image(
            `${args.Name}-base`,
            {
                build: {
                    context: args.Context,
                    dockerfile: args.Dockerfile,
                    platform: 'linux/amd64',
                },
                skipPush: true,
                imageName: pulumi.interpolate`${args.Repo.repositoryUrl}:${args.TagBase}-base`,
            },
            //TMPL {{- if .BaseImage.Raw }}
            {
                dependsOn: pullBaseImage,
            }
            //TMPL {{- end }}
        )

        const sha256 = new command.local.Command(
            `${args.Name}-base-get-sha256-${Date.now()}`,
            { create: pulumi.interpolate`docker image inspect -f ~~{{.ID}} ${base.imageName}` },
            { parent: base }
        ).stdout.apply((id) => id.substring(7))

        return new docker.Image(
            args.Name,
            {
                build: {
                    context: args.Context,
                    dockerfile: args.Dockerfile,
                    platform: 'linux/amd64',
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
                imageName: pulumi.interpolate`${args.Repo.repositoryUrl}:${args.TagBase}-${sha256}`,
            },
            { parent: base }
        )
    })()
}
