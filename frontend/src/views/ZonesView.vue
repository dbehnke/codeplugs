<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useCodeplugStore, type Zone, type Channel } from '../stores/codeplug'

const store = useCodeplugStore()

const selectedZoneId = ref<number | null>(null)
const selectedZone = ref<Zone | null>(null)

// Multi-select state
const selectedAvailableChIDs = ref(new Set<number>())
const selectedMemberIndices = ref(new Set<number>())
let lastSelectedAvailableIndex = -1
let lastSelectedMemberIndex = -1

const isCreating = ref(false)

onMounted(async () => {
   await store.fetchChannels()
   await store.fetchZones()
   if (store.zones.length > 0 && !selectedZoneId.value) {
       selectZone(store.zones[0])
   }
})

const selectZone = (z: Zone) => {
    selectedZoneId.value = z.ID
    // Deep clone to avoid mutating store directly until save
    selectedZone.value = JSON.parse(JSON.stringify(z))
    isCreating.value = false
    selectedMemberIndices.value = new Set()
    lastSelectedMemberIndex = -1
    selectedAvailableChIDs.value = new Set()
    lastSelectedAvailableIndex = -1
}

const createNewZone = () => {
    selectedZone.value = { ID: 0, Name: 'New Zone', Channels: [] }
    selectedZoneId.value = 0
    isCreating.value = true
    selectedMemberIndices.value = new Set()
    lastSelectedMemberIndex = -1
    selectedAvailableChIDs.value = new Set()
    lastSelectedAvailableIndex = -1
}

const availableChannels = computed(() => {
    if (!selectedZone.value) return []
    const memberIDs = new Set(selectedZone.value.Channels.map(c => c.ID))
    return store.channels.filter(c => !memberIDs.has(c.ID))
})

// --- Selection Logic ---

const selectAvailable = (ch: Channel, index: number, event: MouseEvent) => {
    const newSet = new Set(selectedAvailableChIDs.value)

    if (event.shiftKey && lastSelectedAvailableIndex !== -1) {
        const start = Math.min(lastSelectedAvailableIndex, index)
        const end = Math.max(lastSelectedAvailableIndex, index)
        
        if (!event.ctrlKey && !event.metaKey) {
            newSet.clear()
        }
        
        for (let i = start; i <= end; i++) {
            const item = availableChannels.value[i]
            if (item) newSet.add(item.ID)
        }
    } else if (event.ctrlKey || event.metaKey) {
        if (newSet.has(ch.ID)) newSet.delete(ch.ID)
        else newSet.add(ch.ID)
        lastSelectedAvailableIndex = index
    } else {
        newSet.clear()
        newSet.add(ch.ID)
        lastSelectedAvailableIndex = index
    }
    selectedAvailableChIDs.value = newSet
}

const selectMember = (index: number, event: MouseEvent) => {
     const newSet = new Set(selectedMemberIndices.value)

    if (event.shiftKey && lastSelectedMemberIndex !== -1) {
        const start = Math.min(lastSelectedMemberIndex, index)
        const end = Math.max(lastSelectedMemberIndex, index)
        
        if (!event.ctrlKey && !event.metaKey) {
            newSet.clear()
        }
        
        for (let i = start; i <= end; i++) {
            newSet.add(i)
        }
    } else if (event.ctrlKey || event.metaKey) {
        if (newSet.has(index)) newSet.delete(index)
        else newSet.add(index)
        lastSelectedMemberIndex = index
    } else {
        newSet.clear()
        newSet.add(index)
        lastSelectedMemberIndex = index
    }
    selectedMemberIndices.value = newSet
}


// Actions
const addToZone = () => {
    if (selectedAvailableChIDs.value.size > 0 && selectedZone.value) {
        const toAdd: Channel[] = []
        // Maintain order
        availableChannels.value.forEach(ch => {
            if (selectedAvailableChIDs.value.has(ch.ID)) {
                toAdd.push(ch)
            }
        })

        if (toAdd.length > 0) {
            selectedZone.value.Channels.push(...toAdd)
            selectedAvailableChIDs.value = new Set()
            lastSelectedAvailableIndex = -1
        }
    }
}

const removeFromZone = () => {
    if (selectedMemberIndices.value.size > 0 && selectedZone.value) {
        const indices = Array.from(selectedMemberIndices.value).sort((a, b) => b - a)
        indices.forEach(idx => {
            if (idx >= 0 && idx < selectedZone.value!.Channels.length) {
                selectedZone.value!.Channels.splice(idx, 1)
            }
        })
        selectedMemberIndices.value = new Set()
        lastSelectedMemberIndex = -1
    }
}

const moveUp = () => {
    if (selectedMemberIndices.value.size !== 1) return
    const idx = Array.from(selectedMemberIndices.value)[0]

    if (idx !== null && idx > 0 && selectedZone.value) {
        const item = selectedZone.value.Channels[idx]
        selectedZone.value.Channels.splice(idx, 1)
        selectedZone.value.Channels.splice(idx - 1, 0, item)
        
        const newSet = new Set<number>()
        newSet.add(idx - 1)
        selectedMemberIndices.value = newSet
        lastSelectedMemberIndex = idx - 1
    }
}

const moveDown = () => {
    if (selectedMemberIndices.value.size !== 1) return
    const idx = Array.from(selectedMemberIndices.value)[0]

    if (idx !== null && selectedZone.value && idx < selectedZone.value.Channels.length - 1) {
        const item = selectedZone.value.Channels[idx]
        selectedZone.value.Channels.splice(idx, 1)
        selectedZone.value.Channels.splice(idx + 1, 0, item)
        
        const newSet = new Set<number>()
        newSet.add(idx + 1)
        selectedMemberIndices.value = newSet
        lastSelectedMemberIndex = idx + 1
    }
}

const saveZone = async () => {
    if (!selectedZone.value) return
    
    // Logic from App.vue: Save Zone then Assign Channels
    try {
        const res = await fetch('/api/zones', {
             method: 'POST',
             headers: { 'Content-Type': 'application/json' },
             body: JSON.stringify({ ID: selectedZone.value.ID, Name: selectedZone.value.Name })
        })
        if (res.ok) {
            const savedZone = await res.json()
            
            const channelIDs = selectedZone.value.Channels.map(c => c.ID)
            const assignRes = await fetch(`/api/zones/assign?id=${savedZone.ID}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(channelIDs)
            })
            
            if (assignRes.ok) {
                await store.fetchZones()
                // Re-select to get fresh state (including real ID if was 0)
                const fresh = store.zones.find(z => z.ID === savedZone.ID)
                if (fresh) selectZone(fresh)
                alert("Zone Saved!")
            }
        }
    } catch (e) {
        alert("Failed to save zone")
        console.error(e)
    }
}

const deleteZone = async (id: number) => {
    if (confirm("Delete Zone?")) {
        try {
             await fetch(`/api/zones?id=${id}`, { method: 'DELETE' })
             await store.fetchZones()
             if (store.zones.length > 0) selectZone(store.zones[0])
             else selectedZone.value = null
        } catch (e) {
            console.error(e)
        }
    }
}
</script>

<template>
  <div class="h-full flex flex-col p-6">
    <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-bold text-slate-100">Zone Management</h1>
        <button @click="createNewZone" class="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-sm font-medium shadow-lg transition-all">New Zone</button>
    </div>

    <div class="flex-1 flex gap-6 overflow-hidden">
        <!-- Sidebar: Zone List -->
        <div class="w-64 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden">
             <div class="p-4 border-b border-slate-800 font-semibold text-slate-400 text-sm">ZONES</div>
             <div class="flex-1 overflow-y-auto">
                 <div v-for="z in store.zones" :key="z.ID"
                      @click="selectZone(z)"
                      class="px-4 py-3 cursor-pointer hover:bg-slate-800/50 transition-colors border-l-2"
                      :class="selectedZoneId === z.ID ? 'border-indigo-500 bg-slate-800/80 text-white' : 'border-transparent text-slate-400'">
                     {{ z.Name }}
                 </div>
             </div>
        </div>

        <!-- Editor Area -->
        <div v-if="selectedZone" class="flex-1 flex flex-col gap-6">
            <!-- Zone Name Editor -->
             <div class="bg-slate-900/50 border border-slate-800 rounded-2xl p-6 flex items-end gap-4">
                 <div class="flex-1">
                     <label class="block text-xs font-medium text-slate-500 mb-1">Zone Name</label>
                     <input v-model="selectedZone.Name" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/50 font-bold" />
                 </div>
                 <div class="flex gap-2">
                     <button @click="saveZone" class="px-6 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg font-medium transition-colors">Save Zone</button>
                     <button v-if="!isCreating" @click="deleteZone(selectedZone.ID)" class="px-4 py-2 bg-red-900/40 hover:bg-red-900/60 text-red-400 rounded-lg font-medium transition-colors">Delete</button>
                 </div>
             </div>

             <!-- Split View -->
             <div class="flex-1 flex gap-4 min-h-0">
                 <!-- Available Channels -->
                 <div class="flex-1 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden">
                      <div class="p-3 bg-slate-900/80 border-b border-slate-800 text-xs font-bold text-slate-500 uppercase">Available Channels</div>
                      <div class="flex-1 overflow-y-auto p-2 space-y-1">
                          <div v-for="(ch, index) in availableChannels" :key="ch.ID"
                               @click="selectAvailable(ch, index, $event)"
                               class="px-3 py-2 rounded-lg cursor-pointer text-sm transition-colors select-none"
                               :class="selectedAvailableChIDs.has(ch.ID) ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:bg-slate-800'">
                               {{ ch.Name }} <span class="opacity-50 text-xs ml-2">{{ ch.RxFrequency }}</span>
                          </div>
                      </div>
                 </div>

                 <!-- Controls -->
                 <div class="flex flex-col justify-center gap-2">
                     <button @click="addToZone" class="p-2 bg-slate-800 hover:bg-slate-700 rounded text-slate-300 transition-colors" title="Add">
                         <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
                     </button>
                     <button @click="removeFromZone" class="p-2 bg-slate-800 hover:bg-slate-700 rounded text-slate-300 transition-colors" title="Remove">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"></polyline></svg>
                     </button>
                 </div>

                 <!-- Members -->
                 <div class="flex-1 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden">
                      <div class="p-3 bg-slate-900/80 border-b border-slate-800 text-xs font-bold text-slate-500 uppercase">Zone Members</div>
                      <div class="flex-1 overflow-y-auto p-2 space-y-1">
                          <div v-for="(ch, idx) in selectedZone.Channels" :key="idx"
                               @click="selectMember(idx, $event)"
                               class="px-3 py-2 rounded-lg cursor-pointer text-sm transition-colors flex justify-between select-none"
                               :class="selectedMemberIndices.has(idx) ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:bg-slate-800'">
                               <span>{{ idx + 1 }}. {{ ch.Name }}</span>
                               <span class="opacity-50 font-mono">{{ ch.RxFrequency }}</span>
                          </div>
                      </div>
                 </div>

                 <!-- Reorder Controls -->
                  <div class="flex flex-col justify-center gap-2">
                     <button @click="moveUp" 
                        class="p-2 bg-slate-800 hover:bg-slate-700 rounded text-slate-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed" 
                        :disabled="selectedMemberIndices.size !== 1"
                        title="Move Up">
                         <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"></line><polyline points="5 12 12 5 19 12"></polyline></svg>
                     </button>
                     <button @click="moveDown" 
                        class="p-2 bg-slate-800 hover:bg-slate-700 rounded text-slate-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed" 
                        :disabled="selectedMemberIndices.size !== 1"
                        title="Move Down">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><polyline points="19 12 12 19 5 12"></polyline></svg>
                     </button>
                 </div>
             </div>
        </div>
        <div v-else class="flex-1 flex items-center justify-center text-slate-500">
            Select a zone to edit or create a new one.
        </div>
    </div>
  </div>
</template>
