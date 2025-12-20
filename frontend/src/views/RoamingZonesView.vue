<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useCodeplugStore, type RoamingZone, type RoamingChannel } from '../stores/codeplug'
import { Plus, Trash2, Save, ChevronRight, ChevronLeft } from 'lucide-vue-next'

const store = useCodeplugStore()
const selectedZoneId = ref<number | null>(null)
const selectedZone = ref<RoamingZone | null>(null)
const selectedAvailableChId = ref<number | null>(null)
const selectedMemberIndex = ref<number | null>(null)
const isCreating = ref(false)

onMounted(async () => {
    await store.fetchRoamingChannels()
    await store.fetchRoamingZones()
    if (store.roamingZones.length > 0 && !selectedZoneId.value) {
        selectZone(store.roamingZones[0])
    }
})

const selectZone = (z: RoamingZone) => {
    selectedZoneId.value = z.ID
    selectedZone.value = JSON.parse(JSON.stringify(z))
    isCreating.value = false
    selectedMemberIndex.value = null
}

const createNewZone = () => {
    selectedZone.value = { ID: 0, Name: 'New Roaming Zone', Channels: [] }
    selectedZoneId.value = 0
    isCreating.value = true
}

const availableChannels = computed(() => {
    if (!selectedZone.value) return []
    const memberIDs = new Set(selectedZone.value.Channels.map(c => c.ID))
    return store.roamingChannels.filter(c => !memberIDs.has(c.ID))
})

const addToList = () => {
    if (selectedAvailableChId.value && selectedZone.value) {
        const ch = store.roamingChannels.find(c => c.ID === selectedAvailableChId.value)
        if (ch) {
            selectedZone.value.Channels.push(ch)
            selectedAvailableChId.value = null
        }
    }
}

const removeFromList = () => {
    if (selectedMemberIndex.value !== null && selectedZone.value) {
        selectedZone.value.Channels.splice(selectedMemberIndex.value, 1)
        selectedMemberIndex.value = null
    }
}

const saveZone = async () => {
    if (!selectedZone.value) return
    try {
        const res = await fetch('/api/roaming/zones', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ID: selectedZone.value.ID, Name: selectedZone.value.Name })
        })
        if (res.ok) {
            const savedZone = await res.json()
            const realID = savedZone.data?.ID || savedZone.ID
            
            const channelIDs = selectedZone.value.Channels.map(c => c.ID)
            await fetch('/api/roaming/zones/assign', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ roaming_zone_id: realID, roaming_channel_ids: channelIDs })
            })
            
            await store.fetchRoamingZones()
            const fresh = store.roamingZones.find(z => z.ID === realID)
            if (fresh) selectZone(fresh)
        }
    } catch (e) {
        console.error("Failed to save roaming zone", e)
    }
}

const deleteZone = async () => {
    if (selectedZoneId.value && confirm("Delete this zone?")) {
        try {
            await fetch(`/api/roaming/zones?id=${selectedZoneId.value}`, { method: 'DELETE' })
            await store.fetchRoamingZones()
            if (store.roamingZones.length > 0) selectZone(store.roamingZones[0])
            else selectedZone.value = null
        } catch (e) {
            console.error(e)
        }
    }
}
</script>

<template>
    <div class="h-full flex flex-col p-6 overflow-hidden">
        <header class="flex items-center justify-between mb-8">
            <div>
                <h1 class="text-3xl font-bold bg-gradient-to-r from-white to-slate-400 bg-clip-text text-transparent">
                    Roaming Zones
                </h1>
                <p class="text-slate-500 text-sm mt-1">Group roaming channels into geographic zones</p>
            </div>
            <button @click="createNewZone" 
                    class="flex items-center gap-2 px-5 py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl font-semibold shadow-lg shadow-indigo-500/20 transition-all active:scale-95">
                <Plus :size="20" />
                New Zone
            </button>
        </header>

        <div class="flex-1 flex gap-6 min-h-0">
            <!-- Sidebar: Zones -->
            <div class="w-72 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden backdrop-blur-sm">
                <div class="p-4 border-b border-slate-800 font-bold text-slate-500 text-xs uppercase tracking-widest">
                    Available Zones
                </div>
                <div class="flex-1 overflow-y-auto">
                    <div v-for="z in store.roamingZones" :key="z.ID"
                         @click="selectZone(z)"
                         class="px-5 py-4 cursor-pointer hover:bg-slate-800/50 transition-all border-l-4 group"
                         :class="selectedZoneId === z.ID ? 'border-indigo-500 bg-indigo-500/5 text-white' : 'border-transparent text-slate-400'">
                        <div class="font-semibold">{{ z.Name }}</div>
                        <div class="text-xs text-slate-600 mt-1">{{ z.Channels.length }} Channels</div>
                    </div>
                </div>
            </div>

            <!-- Editor -->
            <div v-if="selectedZone" class="flex-1 flex flex-col gap-6 min-w-0">
                <div class="bg-slate-900/50 border border-slate-800 rounded-2xl p-6 flex items-end gap-6 backdrop-blur-sm">
                    <div class="flex-1">
                        <label class="block text-xs font-bold text-slate-500 uppercase mb-2">Zone Name</label>
                        <input v-model="selectedZone.Name" 
                               class="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:ring-2 focus:ring-indigo-500/50 transition-all font-bold text-lg" />
                    </div>
                    <div class="flex gap-3">
                        <button @click="saveZone" 
                                class="flex items-center gap-2 px-6 py-3 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-bold transition-all shadow-lg shadow-emerald-900/20">
                            <Save :size="20" />
                            Save Zone
                        </button>
                        <button v-if="!isCreating" @click="deleteZone" 
                                class="p-3 bg-red-900/40 hover:bg-red-900/60 text-red-400 rounded-xl transition-all">
                            <Trash2 :size="20" />
                        </button>
                    </div>
                </div>

                <div class="flex-1 flex gap-4 min-h-0">
                    <!-- Available Pool -->
                    <div class="flex-1 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden backdrop-blur-sm">
                        <div class="p-4 bg-slate-800/30 border-b border-slate-800 text-xs font-bold text-slate-500 uppercase">Available Channels</div>
                        <div class="flex-1 overflow-y-auto p-4 space-y-2">
                            <div v-for="ch in availableChannels" :key="ch.ID"
                                 @click="selectedAvailableChId = ch.ID"
                                 class="px-4 py-3 rounded-xl cursor-pointer transition-all border group"
                                 :class="selectedAvailableChId === ch.ID ? 'bg-indigo-600 border-indigo-400 text-white' : 'bg-slate-950/50 border-slate-800 text-slate-400 hover:border-slate-600'">
                                <div class="font-medium">{{ ch.Name }}</div>
                                <div class="text-xs opacity-60 font-mono">{{ ch.RxFrequency.toFixed(4) }}</div>
                            </div>
                        </div>
                    </div>

                    <!-- Transfer Controls -->
                    <div class="flex flex-col justify-center gap-4">
                        <button @click="addToList" 
                                class="p-4 bg-slate-800 hover:bg-indigo-600 text-slate-300 hover:text-white rounded-2xl transition-all shadow-xl active:scale-90 disabled:opacity-30"
                                :disabled="!selectedAvailableChId">
                            <ChevronRight :size="24" />
                        </button>
                        <button @click="removeFromList" 
                                class="p-4 bg-slate-800 hover:bg-red-600 text-slate-300 hover:text-white rounded-2xl transition-all shadow-xl active:scale-90 disabled:opacity-30"
                                :disabled="selectedMemberIndex === null">
                            <ChevronLeft :size="24" />
                        </button>
                    </div>

                    <!-- Selected Members -->
                    <div class="flex-1 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden backdrop-blur-sm">
                        <div class="p-4 bg-slate-800/30 border-b border-slate-800 text-xs font-bold text-slate-500 uppercase">Zone Members</div>
                        <div class="flex-1 overflow-y-auto p-4 space-y-2">
                            <div v-for="(ch, idx) in selectedZone.Channels" :key="idx"
                                 @click="selectedMemberIndex = idx"
                                 class="px-4 py-3 rounded-xl cursor-pointer transition-all border group flex justify-between items-center"
                                 :class="selectedMemberIndex === idx ? 'bg-indigo-600 border-indigo-400 text-white' : 'bg-slate-950/50 border-slate-800 text-slate-400 hover:border-slate-600'">
                                <div>
                                    <div class="font-medium text-sm">{{ idx + 1 }}. {{ ch.Name }}</div>
                                    <div class="text-xs opacity-60 font-mono">{{ ch.RxFrequency.toFixed(4) }}</div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <div v-else class="flex-1 flex flex-col items-center justify-center text-slate-600 italic">
                <div class="p-8 bg-slate-900/30 border-2 border-dashed border-slate-800 rounded-3xl">
                    Select a zone from the sidebar to begin editing
                </div>
            </div>
        </div>
    </div>
</template>
