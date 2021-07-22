import { RealtimeClient } from '@supabase/realtime-js'

const client = new RealtimeClient(process.env.VUE_APP_SUPABASE_REALTIME_URL)
client.connect()

const ConnectorChanges = client.channel(`realtime:public:connectors`)
const EndpointChanges = client.channel(`realtime:public:endpoints`)
const SyncChanges = client.channel(`realtime:public:syncs`)
const RunChanges = client.channel(`realtime:public:runs`)

ConnectorChanges.subscribe()
EndpointChanges.subscribe()
SyncChanges.subscribe()
RunChanges.subscribe()

export {
    ConnectorChanges,
    EndpointChanges,
    SyncChanges,
    RunChanges
}