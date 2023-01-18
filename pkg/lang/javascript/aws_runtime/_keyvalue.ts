//@ts-nocheck
'use strict'
import _ = require('lodash')

import moment = require('moment')

import { AWSConfig } from './clients'

import DynamoDB = require('aws-sdk/clients/dynamodb')
const docClient = new DynamoDB.DocumentClient(AWSConfig)

let alldMaps: dMap[] = []

import { Entity } from 'electrodb'

const KVStore = new Entity(
    {
        model: {
            entity: 'entry',
            version: '1',
            service: 'store',
        },
        attributes: {
            map_id: {
                type: 'string',
                required: true,
            },
            kv_key: {
                type: 'string',
                required: true,
            },
            kv_value: {
                type: 'any',
            },
            expiration: {
                type: 'number',
            },
        },
        indexes: {
            kv: {
                pk: {
                    field: 'pk',
                    composite: ['map_id'],
                },
                sk: {
                    field: 'sk',
                    composite: ['kv_key'],
                },
            },
        },
        filters: {},
    },
    { table: '{{.AppName}}', client: docClient }
)

interface MapOptions {
    id: string

    /* The next two options change 4 different trade-offs:
        | batch_write | writeOn |                                                                                
        |            | Change  |                                                                               
        +------------+---------+ 
        | TRUE       | TRUE    | Every time a change happens in 
        |            |         | the KV, an SQS update is sent to be batch updated    
        |            |         |  
        | TRUE       | FALSE   | On lambda exit, write final KV
        |            |         | updates to SQS for Batching (no local intermediates) 
        |            |         |  
        | FALSE      | FALSE   | On lambda exit, write final KV
        |            |         | updates to Dynamo (no local intermediates)           
        |            |         |  
        | FALSE      | TRUE    | Every time a change happens in the KV
        |                      | write to Dynamo immediately   
    */
    batch_write: boolean
    write_on_change: boolean

    ttl?: number
    versioned?: boolean
}

export class dMap<V = any> {
    //TODO: Haven't dealt with Pagination yet. Limits are around size - 1MB for queries -
    //Should be quite a bit of text for now - but needs looking into

    private opts: MapOptions

    private _cache: Map<string, any>
    public dynamoCalls = 0
    nonCachedFunctionCalls = 0
    allFunctionCalls = 0
    deletedKeys: string[] = []

    constructor(opts?: Partial<MapOptions>) {
        this.opts = {
            id: '',
            batch_write: false,
            write_on_change: true,
            ...opts,
        }

        this._cache = new Map()

        alldMaps.push(this)
    }

    public emptyCache() {
        this._cache.clear()
    }

    async get<T = V>(key: string): Promise<T>
    async get<T = V>(key: string): Promise<T | undefined>
    async get<T = V>(key: string): Promise<T | undefined> {
        try {
            this.allFunctionCalls += 1

            this.nonCachedFunctionCalls += 1

            let dbValue = (await KVStore.query.kv({ kv_key: key, map_id: this.opts.id }).go()).data

            let value = dbValue?.[0]?.kv_value as any as V
            if ((value as any) == '_DELETED') {
                return undefined
            }
            // value = this._restoreUndefinedValues(value);
            this._cache.set(key, value)

            return value as any as T
        } catch (error) {
            console.error('CloudCC Runtime error')
            console.error(error)
        }
    }

    async has(key: string) {
        return typeof (await this.get(key)) !== 'undefined'
    }

    /**
     *
     * @param key
     * @param value
     * @param ttl time-to-live, seconds
     */
    async set<T = V>(key: string, value: T) {
        if (key == 'options') return this // reserved keyword that functions as the constructor
        //TODO: Need to calculate and manage Deltas to avoid continued growth of what we send
        //      to be batched

        if (key != 'flush') this._cache.set(key, value)

        if (typeof value == 'object' && this.opts.versioned) {
            const v = value as any
            let whereFunc
            const hadVersion = '__version' in v
            if (hadVersion) {
                v.__version++
                whereFunc = ({ kv_value }, { eq }) => eq(kv_value.__version, v.__version - 1)
            } else {
                v.__version = 0
                whereFunc = ({ kv_value }, { notExists }) => notExists(kv_value)
            }
            try {
                await KVStore.put(this.toKVObject(key, v)).where(whereFunc).go()
            } catch (err) {
                if (err.message.includes('conditional request failed')) {
                    if (hadVersion) {
                        throw new Error(
                            `Conditional put failed: expected version ${
                                v.__version - 1
                            } did not match`
                        )
                    } else {
                        throw new Error('Conditional put failed: expected item to not exist')
                    }
                } else {
                    throw err
                }
            }
            return this
        }
        if (this.opts.batch_write == false && this.opts.write_on_change == true) {
            // Every time a change happens in the KV write to Dynamo immediately
            await this.flushEntries([[key, value as any]])
        } else if (
            key == 'flush' &&
            this.opts.batch_write == false &&
            this.opts.write_on_change == false
        ) {
            // On lambda exit, write final KV updates to Dynamo (no local intermediates)
            await this.flushEntries(Array.from(this._cache.entries()))
        }

        return this
    }

    private async flushEntries(entriesToFlush: [string, any][]) {
        const cachedObjects = entriesToFlush.map(([key, value]) => this.toKVObject(key, value))
        try {
            await KVStore.put(cachedObjects).go()
        } catch (e) {
            console.log(e)
        }
    }

    private expiration(): number | undefined {
        if (this.opts.ttl) {
            return moment().add(this.opts.ttl, 'seconds').unix()
        }
        return undefined
    }

    private toKVObject(key: string, value: any) {
        return {
            map_id: this.opts.id,
            kv_key: key,
            kv_value: value,
            expiration: this.expiration(),
        }
    }

    async delete<T = V>(key: string): Promise<boolean | undefined> {
        if (key == 'options') return true // reserved keyword that functions as the constructor
        if (key == 'flush') {
            await this._cache.delete(key)
            return true
        }
        this.deletedKeys.push(key)
        // We don't actually delete keys - bad practice. We only allow clearing the entire set. We filter out the deleted ones later
        await this.set(key, '_DELETED')
        return await this._cache.delete(key)
    }

    async keys<T>(): Promise<string[]> {
        this.allFunctionCalls += 1
        try {
            this.nonCachedFunctionCalls += 1
            this.dynamoCalls += 1
            const keyResults = (await KVStore.query.kv({ map_id: this.opts.id }).go()).data

            const filteredKeys = _.uniq([
                ...keyResults.filter((x) => x.kv_value != '_DELETED').map((x) => x.kv_key),
                ...this._cache.keys(),
            ])
            this.deletedKeys.forEach((key) => {
                _.remove(filteredKeys, (x) => x == key)
            })

            return filteredKeys.map((x) => x)
        } catch (e) {
            console.error(e)
            throw new Error(`CloudCompiler runtime error:`)
        }
    }

    async entries(): Promise<[string, any][]> {
        this.nonCachedFunctionCalls += 1
        this.dynamoCalls += 1

        try {
            let keyResults = (await KVStore.query.kv({ map_id: this.opts.id }).go()).data

            this.deletedKeys.forEach((key) => {
                _.remove(keyResults, (x) => x.kv_key == key)
            })

            keyResults = keyResults.filter((x) => x.kv_value != '_DELETED') //.map(x => [x.kv_key, this._restoreUndefinedValues(x.kv_value)])

            keyResults.map((kvPair) => {
                if (this._cache.has(kvPair.kv_key)) return
                this._cache.set(kvPair.kv_key, kvPair.kv_value)
            })

            return this._cache.entries() as any
        } catch (e) {
            console.error(e)
            throw new Error(`CloudCompiler runtime error:`)
        }
    }

    async clear<T = V>(): Promise<boolean | undefined> {
        try {
            let entries = await KVStore.query.kv({ map_id: this.opts.id }).go()
            let result = await KVStore.delete(entries.data).go()
            this._cache.clear()
            return true
        } catch (err) {
            console.log(err)
            return undefined
        }
    }
}
