//@ts-nocheck
'use strict'

export function getParams(
    hostEnvVarName: string,
    portEnvVarName: string,
    params: { [key: string]: any }
): dict {
    let newParams = {
        ...params,
        socket: {
            ...params['socket'],
            host: process.env[hostEnvVarName],
            port: process.env[portEnvVarName],
        },
    }
    return {
        ...newParams,
    }
}
