//@ts-nocheck
'use strict'

export function getParams(dbName: string, params: { [key: string]: any }): dict {
    let newParams = {
        ...params,
        socket: {
            ...params['socket'],
            host: process.env[`${dbName.toUpperCase()}_PERSIST_REDIS_NODE_HOST`],
            port: process.env[`${dbName.toUpperCase()}_PERSIST_REDIS_NODE_PORT`],
        },
    }
    return {
        ...newParams,
    }
}
