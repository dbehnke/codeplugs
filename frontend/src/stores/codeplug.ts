import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface Channel {
    ID: number
    Name: string
    RxFrequency: number
    TxFrequency: number
    Mode: string // 'FM', 'NFM', 'DMR'
    Tone: string
    Skip: boolean
    SquelchType: string
    RxTone: string
    TxTone: string
    RxDCS: string
    TxDCS: string
    Type: string // 'Analog', 'Digital (DMR)', 'Digital (NXDN)'
    Protocol: string // 'FM', 'NFM', 'DMR'
    ColorCode: number
    TimeSlot: number
    ContactID?: number
    Power: string // 'High', 'Mid', 'Low', 'Turbo'
    Bandwidth: string // '12.5', '25'
    ScanList: string
    RxGroup: string
    TxContact: string // For display or legacy
    TalkAround: boolean
    WorkAlone: boolean
    TxPermit: string // 'Always', 'ChannelFree', 'ColorCode'
    RxSquelchMode: string // 'Normal', 'Strict'
    Notes: string
}

export interface Contact {
    ID: number
    Name: string
    Callsign: string
    City: string
    State: string
    Country: string
    Remarks: string
    DMRID: number
    Type: string // 'Group', 'Private', 'All Call'
    Source: string // 'User', 'RadioID'
}

export interface Zone {
    ID: number
    Name: string
    Channels: Channel[]
}

export interface ScanList {
    ID: number
    Name: string
    Channels: Channel[]
}

export const useCodeplugStore = defineStore('codeplug', () => {
    // State
    const channels = ref<Channel[]>([])
    const zones = ref<Zone[]>([])
    const scanlists = ref<ScanList[]>([])
    const talkgroups = ref<Contact[]>([]) // User contacts
    const dmrContacts = ref<Contact[]>([]) // RadioID contacts

    // Loading States
    const loadingChannels = ref(false)
    const loadingZones = ref(false)
    const loadingScanLists = ref(false)
    const loadingTalkgroups = ref(false)
    const loadingDMRContacts = ref(false)

    // Actions
    async function fetchChannels() {
        loadingChannels.value = true
        try {
            const res = await fetch('/api/channels')
            channels.value = await res.json()
        } catch (e) {
            console.error("Failed to fetch channels", e)
        } finally {
            loadingChannels.value = false
        }
    }

    async function fetchZones() {
        loadingZones.value = true
        try {
            const res = await fetch('/api/zones')
            zones.value = await res.json()
        } catch (e) {
            console.error("Failed to fetch zones", e)
        } finally {
            loadingZones.value = false
        }
    }

    async function fetchScanLists() {
        loadingScanLists.value = true
        try {
            const res = await fetch('/api/scanlists')
            scanlists.value = await res.json()
        } catch (e) {
            console.error("Failed to fetch scan lists", e)
        } finally {
            loadingScanLists.value = false
        }
    }

    async function fetchTalkgroups() {
        loadingTalkgroups.value = true
        try {
            const res = await fetch('/api/contacts?source=User&limit=1000')
            const data = await res.json()
            talkgroups.value = data.data || []
        } catch (e) {
            console.error("Failed to fetch talkgroups", e)
        } finally {
            loadingTalkgroups.value = false
        }
    }

    // Helper for pagination (mostly handled in component for now, but store keeps data)
    async function fetchDMRContacts(page = 1, limit = 50, search = '', sort = 'name', order = 'asc') {
        loadingDMRContacts.value = true
        try {
            const params = new URLSearchParams({
                source: 'RadioID',
                page: page.toString(),
                limit: limit.toString(),
                search,
                sort,
                order
            })
            const res = await fetch(`/api/contacts?${params.toString()}`)
            const data = await res.json()
            dmrContacts.value = data.data || []
            return data.meta // Return meta for component to handle total pages
        } catch (e) {
            console.error("Failed to fetch digital contacts", e)
            return { total: 0 }
        } finally {
            loadingDMRContacts.value = false
        }
    }

    async function deleteChannel(id: number) {
        try {
            await fetch(`/api/channels?id=${id}`, { method: 'DELETE' })
            await fetchChannels()
        } catch (e) {
            console.error("Failed to delete channel", e)
            throw e
        }
    }

    async function saveChannel(channel: Channel) {
        try {
            const res = await fetch('/api/channels', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(channel)
            })
            if (!res.ok) throw new Error("Failed to save")
            await fetchChannels()
        } catch (e) {
            console.error("Failed to save channel", e)
            throw e
        }
    }

    return {
        channels,
        zones,
        scanlists,
        talkgroups,
        dmrContacts,
        loadingChannels,
        loadingZones,
        loadingScanLists,
        loadingTalkgroups,
        loadingDMRContacts,
        fetchChannels,
        fetchZones,
        fetchScanLists,
        fetchTalkgroups,
        fetchDMRContacts,
        deleteChannel,
        saveChannel
    }
})
