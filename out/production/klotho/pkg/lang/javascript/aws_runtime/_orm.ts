//@ts-nocheck
'use strict'

const ormPrefix = '{{.AppName}}'

export function getDBConn(dbNameEnvVar: string): string {
    const conn = process.env[dbNameEnvVar]
    return conn
}

export function getDataSourceParams(dbNameEnvVar: string, params: { [key: string]: number }): dict {
    let newParams = { ...params }
    const fieldsToDelete = ['host', 'type', 'port', 'username', 'passowrd', 'database']
    for (const field of fieldsToDelete) {
        delete newParams[field]
    }

    return {
        ...newParams,
        type: 'postgres',
        url: getDBConn(dbNameEnvVar),
    }
}
