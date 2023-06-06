import * as fs from 'fs'
import * as os from 'os'
import * as path from 'path'
import * as requestRetry from 'requestretry'

const analyticsFile = '/.klotho/analytics.json'
const klothoServer = 'http://srv.klo.dev/analytics/track'

export interface Analytics {
    id: string
    event: string
    properties?: any
}

export interface User {
    id?: string
    email?: string
}

export const sendAnalytics = async (user: User, message: string, appName: string) => {
    let id = ''
    const properties = { _logLevel: 'info', app: appName }
    if (user.id != undefined) {
        id = user.id
    }
    if (user.email != undefined) {
        id = user.email
    }

    if (id != '') {
        const data: Analytics = {
            id: id,
            event: message,
            properties: properties,
        }

        const resp = await requestRetry({
            url: klothoServer,
            method: 'POST',
            json: true,
            body: data,
            maxAttempts: 3,
            retryDelay: 100,
        })
    }
}

export const retrieveUser = async (): Promise<User> => {
    const file = path.join(os.homedir(), analyticsFile)
    if (fs.existsSync(file)) {
        const data = await fs.promises.readFile(file, 'utf-8')
        const user: User = JSON.parse(data)
        return user
    }
    return {}
}
