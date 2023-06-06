//@ts-nocheck
'use strict'

import { GetSecretValueCommand, SecretsManagerClient } from '@aws-sdk/client-secrets-manager'

const client = new SecretsManagerClient({})

const secretPrefix = '{{.AppName}}'

export async function readFile(path: string): Promise<Buffer> {
    try {
        const cmd = new GetSecretValueCommand({ SecretId: `${secretPrefix}-${path}` })

        const data = await client.send(cmd)

        if (data.SecretBinary) {
            return Buffer.from(data.SecretBinary)
        }
        if (data.SecretString) {
            return Buffer.from(data.SecretString, 'utf-8')
        }
        throw new Error(`Empty secret for ${path}`)
    } catch (err) {
        throw new Error(`Could not read secret '${path}'`)
    }
}
