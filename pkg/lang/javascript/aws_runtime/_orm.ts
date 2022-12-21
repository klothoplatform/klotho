//@ts-nocheck
'use strict'

const ormPrefix = '{{.AppName}}'

export function getDBConn(dbName: string): string {
    const conn = process.env[`${dbName.toUpperCase()}_PERSIST_ORM_CONNECTION`]
    return conn
}

export function getDataSourceParams(dbName: string, params: { [key: string]: number }): dict {
    let newParams = { ...params }
    const fieldsToDelete = ['host', 'type', 'port', 'username', 'passowrd', 'database']
    for (const field of fieldsToDelete) {
        delete newParams[field]
    }

    return {
        ...newParams,
        type: 'postgres',
        url: getDBConn(dbName),
    }
}
