import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'
import * as inputs from '@pulumi/aws/types/input'

interface Args {
    Name: string
    CertificateTransparencyLoggingPreference?: string
    DomainName: string
    EarlyRenewalDuration?: string
    SubjectAlternativeNames?: string[]
    Tags: ModelCaseWrapper<Record<string, string>>
    ValidationMethod?: string
    DomainValidationOptions?: pulumi.Input<pulumi.Input<inputs.acm.CertificateValidationOption>[]>
}

function create(args: Args): aws.acm.Certificate {
    return new aws.acm.Certificate(args.Name, {
        //TMPL {{- if .CertificateTransparencyLoggingPreference }}
        options: {
            certificateTransparencyLoggingPreference: args.CertificateTransparencyLoggingPreference,
        },
        //TMPL {{- end }}
        //TMPL {{- if .EarlyRenewalDuration }}
        earlyRenewalDuration: args.EarlyRenewalDuration,
        //TMPL {{- end }}
        //TMPL {{- if .SubjectAlternativeNames }}
        subjectAlternativeNames: args.SubjectAlternativeNames,
        //TMPL {{- end }}
        //TMPL {{- if .ValidationMethod }}
        validationMethod: args.ValidationMethod,
        //TMPL {{- end }}
        //TMPL {{- if .DomainValidationOptions }}
        validationOptions: args.DomainValidationOptions,
        //TMPL {{- end }}
        domainName: args.DomainName,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.acm.Certificate, args: Args) {
    return {
        Arn: object.arn,
    }
}
