//@ts-nocheck
'use strict'

// The cluster client requires root nodes and defaults to be able to properly connect and get redirected to new slots in memorydb
export function getParams(
    hostEnvVarName: string,
    portEnvVarName: string,
    params: { [key: string]: any }
): dict {
    const socketDefaults = {}
    if (params['defaults']?.socket) {
        socketDefaults = params['defaults'].socket
    }
    let newParams = {
        ...params,
        rootNodes: [
            {
                socket: {
                    host: `${process.env[hostEnvVarName]}`,
                    port: `${process.env[portEnvVarName]}`,
                    tls: true,
                },
            },
        ],
        defaults: {
            ...params['defaults'],
            socket: {
                ...socketDefaults,
                host: `${process.env[hostEnvVarName]}`,
                port: `${process.env[portEnvVarName]}`,
                tls: true,
            },
        },
    }
    return {
        ...newParams,
    }
}
