import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper, TemplateWrapper } from '../../wrappers'

interface Args {
    Name: string
    Origins: aws.types.input.cloudfront.DistributionOrigin[]
    ViewerCertificate: TemplateWrapper<aws.types.input.cloudfront.DistributionViewerCertificate>
    Enabled: boolean
    DefaultCacheBehavior: aws.types.input.cloudfront.DistributionDefaultCacheBehavior
    CacheBehaviors: aws.types.input.cloudfront.DistributionCacheBehavior[]
    Restrictions: aws.types.input.cloudfront.DistributionRestrictions
    DefaultRootObject: string
    Aliases: string[]
    CustomErrorResponses: aws.types.input.cloudfront.DistributionCustomErrorResponse[]
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.cloudfront.Distribution {
    return new aws.cloudfront.Distribution(args.Name, {
        origins: args.Origins,
        enabled: args.Enabled,
        viewerCertificate: args.ViewerCertificate,
        orderedCacheBehaviors: args.CacheBehaviors,
        //TMPL {{- if .Aliases }}
        aliases: args.Aliases,
        //TMPL {{- end }}
        //TMPL {{- if .CustomErrorResponses }}
        customErrorResponses: args.CustomErrorResponses,
        //TMPL {{- end }}
        //TMPL {{- if (index .DefaultCacheBehavior "targetOriginId") }}
        defaultCacheBehavior: args.DefaultCacheBehavior,
        //TMPL {{- else }}
        //TMPL defaultCacheBehavior: {
        //TMPL     ...args.DefaultCacheBehavior,
        //TMPL     targetOriginId: {{(index .Origins 0).originId}},
        //TMPL },
        //TMPL {{- end }}
        restrictions: args.Restrictions,
        //TMPL {{- if .DefaultRootObject }}
        defaultRootObject: args.DefaultRootObject,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: ReturnType<typeof create>, args: Args) {
    return {
        DomainName: object.domainName,
        URLBase: pulumi.interpolate`https://${object.domainName}`,
    }
}

function infraExports(
    object: ReturnType<typeof create>,
    args: Args,
    props: ReturnType<typeof properties>
) {
    return {
        Domain: object.domainName,
    }
}
