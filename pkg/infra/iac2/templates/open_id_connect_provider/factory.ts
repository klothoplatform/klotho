import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { getIssuerCAThumbprint } from '@pulumi/eks/cert-thumprint'
import * as https from 'https'

interface Args {
    Name: string
    ClientIdLists: string[]
    Cluster: aws.eks.Cluster
    Region: pulumi.Output<pulumi.UnwrappedObject<aws.GetRegionResult>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.iam.OpenIdConnectProvider {
    return new aws.iam.OpenIdConnectProvider(`oidcProvider`, {
        clientIdLists: ['sts.amazonaws.com'],
        url: args.Cluster.identities[0].oidcs[0].issuer,
        thumbprintLists: [
            getIssuerCAThumbprint(
                pulumi.interpolate`https://oidc.eks.${args.Region.name}.amazonaws.com`,
                new https.Agent({
                    // Cached sessions can result in the certificate not being
                    // available since its already been "accepted." Disable caching.
                    maxCachedSessions: 0,
                })
            ),
        ],
    })
}
