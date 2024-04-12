import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    AvailabilityZones: string[]
    CapacityRebalance: boolean
    Cooldown: string
    DesiredCapacity: string
    DesiredCapacityType: string
    HealthCheckGracePeriod: number
    InstanceId: string
    LaunchTemplate: ModelCaseWrapper<Record<string, pulumi.Input<string>>>
    MaxSize: string
    MinSize: string
    VPCZoneIdentifier: string[]
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.autoscaling.Group {
    return new aws.autoscaling.Group(args.Name, {
        //TMPL {{- if .AvailabilityZones }}
        availabilityZones: args.AvailabilityZones,
        //TMPL {{- end }}
        //TMPL {{- if .CapacityRebalance }}
        capacityRebalance: args.CapacityRebalance,
        //TMPL {{- end }}
        //TMPL {{- if .Cooldown }}
        defaultCooldown: args.Cooldown,
        //TMPL {{- end }}
        //TMPL {{- if .DesiredCapacity }}
        desiredCapacity: args.DesiredCapacity,
        //TMPL {{- end }}
        //TMPL {{- if .DesiredCapacityType }}
        desiredCapacityType: args.DesiredCapacityType,
        //TMPL {{- end }}
        //TMPL {{- if .HealthCheckGracePeriod }}
        healthCheckGracePeriod: args.HealthCheckGracePeriod,
        //TMPL {{- end }}
        //TMPL {{- if .InstanceId }}
        instanceId: args.InstanceId,
        //TMPL {{- end }}
        launchTemplate: {
            //TMPL {{- if .LaunchTemplate.LaunchTemplateId }}
            //TMPL id:  {{ .LaunchTemplate.LaunchTemplateId }},
            //TMPL {{- end }}
            //TMPL {{- if .LaunchTemplate.LaunchTemplateName }}
            //TMPL name: {{ .LaunchTemplate.LaunchTemplateName }},
            //TMPL {{- end }}
            //TMPL {{- if .LaunchTemplate.Version }}
            //TMPL version: {{ .LaunchTemplate.Version }},
            //TMPL {{- end }}
        },
        tags: [
            //TMPL {{- range $key, $value := .Tags }}
            //TMPL    {
            //TMPL        key: "{{ $key }}",
            //TMPL        value: {{ $value }},
            //TMPL        propagateAtLaunch: true,
            //TMPL    },
            //TMPL{{- end }}
        ],
        maxSize: args.MaxSize,
        minSize: args.MinSize,
        vpcZoneIdentifiers: args.VPCZoneIdentifier,
    })
}

function properties(object: aws.autoscaling.Group, args: Args) {
    return {
        Arn: object.arn,
        Id: object.id,
    }
}

function importResource(args: Args): aws.autoscaling.Group {
    return aws.autoscaling.Group.get(args.Name, args.Id)
}
