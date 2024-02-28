import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import { ModelCaseWrapper } from '../../wrappers'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    AccessString: string
    AuthenticationMode: awsInputs.memorydb.UserAuthenticationMode
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.memorydb.User {
    return new aws.memorydb.User(args.Name, {
        accessString: args.AccessString,
        authenticationMode: {
            type: 'password',
            passwords: [kloConfig.requireSecret(`${args.Name}-password`)],
        },
        userName: args.Name,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.memorydb.User, args: Args) {
    return {
        Arn: object.arn,
        Id: object.id,
        Username: object.userName,
        Password: kloConfig.requireSecret(`${args.Name}-password`),
    }
}
