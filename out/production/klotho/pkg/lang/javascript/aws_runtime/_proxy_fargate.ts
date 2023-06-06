const axios = require('axios')
import { ServiceDiscoveryClient, DiscoverInstancesCommand } from '@aws-sdk/client-servicediscovery'

const { APP_NAME } = process.env

export async function proxyCall(
    callType: string,
    execGroupName: string,
    moduleName: string,
    functionToCall: string,
    params: any[]
) {
    try {
        const hostname = await getExecFargateInstance(execGroupName)
        const res = await axios({
            method: 'post',
            url: `http://${hostname}:3001`,
            data: {
                callType,
                execGroupName,
                functionToCall,
                moduleName,
                params,
            },
        })
        return res.data
    } catch (error) {
        console.log(error)
        throw error
    }
}

async function getExecFargateInstance(logicalName): Promise<string> {
    try {
        const client = new ServiceDiscoveryClient({})
        const command = new DiscoverInstancesCommand({
            NamespaceName: `${APP_NAME}-privateDns`,
            ServiceName: logicalName,
        })
        const response = await client.send(command)

        const ips = response.Instances?.reduce((ips, instance): string[] => {
            const ip = instance.Attributes?.AWS_INSTANCE_IPV4
            if (ip) {
                ips.push(ip)
            }
            return ips
        }, [] as string[])

        if (ips == undefined || ips.length == 0) {
            throw new Error(`No IPs found for ${logicalName}`)
        }

        return ips[0]
    } catch (e) {
        console.log(e)
        throw e
    }
}
