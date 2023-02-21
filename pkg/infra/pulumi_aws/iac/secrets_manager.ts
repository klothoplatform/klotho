import { CloudCCLib, Resource } from '../deploylib'
import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

import * as fs from 'fs'

import { hash as h, validate } from './sanitization/sanitizer'
import AwsSanitizer from './sanitization/aws'

export interface Secret {
    Name: string
    FilePath: string
    Params: any
}

export class SecretsManager {
    constructor(private lib: CloudCCLib, secrets: Secret[]) {
        for (const secret of secrets) {
            if (secret.FilePath == '') {
                secret.FilePath = this.getSecretFilePath(secret.Name)
            }
            if (secret.FilePath != '') {
                this.setupSecretFromFile(secret)
            } else {
                throw new Error('Unsupported value type for secret')
            }
        }
    }

    private getSecretFilePath(secretName: string): string {
        const config = new pulumi.Config()
        const valuesFile = config.require(`config-${secretName}-FilePath`)
        return valuesFile
    }

    private setupSecretFromFile(secret: Secret) {
        const secretName = `${this.lib.name}-${secret.Name}`
        validate(secretName, AwsSanitizer.SecretsManager.secret.nameValidation())
        let awsSecret: aws.secretsmanager.Secret
        if (this.lib.secrets.has(secret.Name)) {
            awsSecret = this.lib.secrets.get(secret.Name)!
        } else {
            awsSecret = new aws.secretsmanager.Secret(
                `${secret.Name}`,
                {
                    name: secretName,
                    recoveryWindowInDays: 0,
                },
                { protect: this.lib.protect }
            )
            if (fs.existsSync(secret.FilePath)) {
                new aws.secretsmanager.SecretVersion(
                    `${secret.Name}`,
                    {
                        secretId: awsSecret.id,
                        secretBinary: fs.readFileSync(secret.FilePath).toString('base64'),
                    },
                    { protect: this.lib.protect }
                )
            }
            this.lib.secrets.set(secret.Name, awsSecret)
        }
        this.addPermissions(awsSecret.arn)
    }

    private addPermissions(secretArn: pulumi.Output<string>) {
        this.lib.topology.topologyIconData.forEach((resource) => {
            if (
                resource.kind == Resource.secret ||
                (resource.kind == Resource.config && resource.type == 'secrets_manager')
            ) {
                this.lib.topology.topologyEdgeData.forEach((edge) => {
                    if (edge.target == resource.id) {
                        this.lib.addPolicyStatementForName(
                            this.lib.resourceIdToResource.get(edge.source).title,
                            {
                                Effect: 'Allow',
                                Action: [
                                    'secretsmanager:GetSecretValue',
                                    'secretsmanager:DescribeSecret',
                                ],
                                Resource: [secretArn],
                            }
                        )
                    }
                })
            }
        })
    }
}
