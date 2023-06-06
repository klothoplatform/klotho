import * as express from 'express'
import * as path from 'path'

//TMPL {{if .Expose.AppModule}}
require('../{{.Expose.AppModule}}')
//TMPL {{end}}

//TMPL {{if .MainModule}}
require('../{{.MainModule}}')
//TMPL {{end}}

const app = express()
const port = 3001

app.use(express.json())

interface RPCParams {
    callType: string
    execGroupName: string
    functionToCall: string
    moduleName: string
    params: any[]
}

app.get('/', (req: express.Request, res: express.Response) => {
    res.sendStatus(200)
})

app.post('/', async (req: express.Request, res: express.Response) => {
    const params: RPCParams = req.body
    console.info(`Dispatched:`, params)

    try {
        const mode = parseMode(params.callType)

        switch (mode) {
            case 'rpc':
                const result = await require(path.join('../', params.moduleName))[
                    params.functionToCall
                ].apply(null, params.params)
                res.send(result)
                break
        }
    } catch (err) {
        console.error(`Dispatcher Failed`, err)
        throw err
    }
})

function parseMode(__callType: string): string | undefined {
    if (__callType === 'rpc') return 'rpc'
}

app.listen(port, () => {
    console.log(`Klotho RPC Proxy listening on: ${port}`)
})
