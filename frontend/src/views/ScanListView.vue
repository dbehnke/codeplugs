<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useCodeplugStore, type ScanList, type Channel } from '../stores/codeplug'

const store = useCodeplugStore()

const selectedListId = ref<number | null>(null)
const selectedList = ref<ScanList | null>(null)
const selectedAvailableChId = ref<number | null>(null)
const selectedMemberIndex = ref<number | null>(null)
const isCreating = ref(false)

// Delete Confirm Logic
const showDeleteConfirm = ref(false)
const listToDelete = ref<number | null>(null)

onMounted(async () => {
    await store.fetchChannels()
    await store.fetchScanLists()
    if (store.scanlists.length > 0 && !selectedListId.value) {
        selectList(store.scanlists[0])
    }
})

const selectList = (z: ScanList) => {
    selectedListId.value = z.ID
    // Deep clone to avoid mutating store directly until save
    selectedList.value = JSON.parse(JSON.stringify(z))
    isCreating.value = false
    selectedMemberIndex.value = null
}

const createNewList = () => {
    selectedList.value = { ID: 0, Name: 'New Scan List', Channels: [] }
    selectedListId.value = 0
    isCreating.value = true
}

const availableChannels = computed(() => {
    if (!selectedList.value) return []
    const memberIDs = new Set(selectedList.value.Channels.map(c => c.ID))
    return store.channels.filter(c => !memberIDs.has(c.ID))
})

// Actions
const addToList = () => {
    if (selectedAvailableChId.value && selectedList.value) {
         // Prevent Adding duplicates
        if (selectedList.value.Channels.some(c => c.ID === selectedAvailableChId.value)) {
            selectedAvailableChId.value = null
            return
        }
        
        const ch = store.channels.find(c => c.ID === selectedAvailableChId.value)
        if (ch) {
            selectedList.value.Channels.push(ch)
            selectedAvailableChId.value = null
        }
    }
}

const removeFromList = () => {
    if (selectedMemberIndex.value !== null && selectedList.value) {
        selectedList.value.Channels.splice(selectedMemberIndex.value, 1)
        selectedMemberIndex.value = null
    }
}

const moveUp = () => {
    const idx = selectedMemberIndex.value
    if (idx !== null && idx > 0 && selectedList.value) {
        const item = selectedList.value.Channels[idx]
        selectedList.value.Channels.splice(idx, 1)
        selectedList.value.Channels.splice(idx - 1, 0, item)
        selectedMemberIndex.value = idx - 1
    }
}

const moveDown = () => {
    const idx = selectedMemberIndex.value
    if (idx !== null && selectedList.value && idx < selectedList.value.Channels.length - 1) {
        const item = selectedList.value.Channels[idx]
        selectedList.value.Channels.splice(idx, 1)
        selectedList.value.Channels.splice(idx + 1, 0, item)
        selectedMemberIndex.value = idx + 1
    }
}

const saveList = async () => {
    if (!selectedList.value) return
    
    // Logic: Save List then Assign Channels
    try {
        const res = await fetch('/api/scanlists', {
             method: 'POST',
             headers: { 'Content-Type': 'application/json' },
             body: JSON.stringify({ ID: selectedList.value.ID, Name: selectedList.value.Name })
        })
        if (res.ok) {
            const savedList = await res.json()
            
            const channelIDs = selectedList.value.Channels.map(c => c.ID)
            const assignRes = await fetch('/api/scanlists/assign', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ scan_list_id: savedList.ID, channel_ids: channelIDs })
            })
            
            if (assignRes.ok) {
                await store.fetchScanLists()
                // Re-select to get fresh state (including real ID if was 0)
                const fresh = store.scanlists.find(z => z.ID === savedList.ID)
                if (fresh) selectList(fresh)
                // alert("Scan List Saved!") // Removed alert for cleaner UI
            }
        }
    } catch (e) {
        alert("Failed to save scan list")
        console.error(e)
    }
}

const confirmDelete = (id: number) => {
    listToDelete.value = id
    showDeleteConfirm.value = true
}

const deleteList = async () => {
    if (listToDelete.value !== null) {
        try {
             await fetch(`/api/scanlists?id=${listToDelete.value}`, { method: 'DELETE' })
             await store.fetchScanLists()
             if (store.scanlists.length > 0) selectList(store.scanlists[0])
             else selectedList.value = null
        } catch (e) {
            console.error(e)
        }
    }
    showDeleteConfirm.value = false
    listToDelete.value = null
}
</script>

<template>
  <div class="h-full flex flex-col p-6">
    <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-bold text-slate-100">Scan List Management</h1>
        <button @click="createNewList" class="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-sm font-medium shadow-lg transition-all">New Scan List</button>
    </div>

    <div class="flex-1 flex gap-6 overflow-hidden">
        <!-- Sidebar: List -->
        <div class="w-64 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden">
             <div class="p-4 border-b border-slate-800 font-semibold text-slate-400 text-sm">SCAN LISTS</div>
             <div class="flex-1 overflow-y-auto">
                 <div v-for="z in store.scanlists" :key="z.ID"
                      @click="selectList(z)"
                      class="px-4 py-3 cursor-pointer hover:bg-slate-800/50 transition-colors border-l-2"
                      :class="selectedListId === z.ID ? 'border-indigo-500 bg-slate-800/80 text-white' : 'border-transparent text-slate-400'">
                     {{ z.Name }}
                 </div>
             </div>
        </div>

        <!-- Editor Area -->
        <div v-if="selectedList" class="flex-1 flex flex-col gap-6">
            <!-- Name Editor -->
             <div class="bg-slate-900/50 border border-slate-800 rounded-2xl p-6 flex items-end gap-4">
                 <div class="flex-1">
                     <label class="block text-xs font-medium text-slate-500 mb-1">List Name</label>
                     <input v-model="selectedList.Name" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/50 font-bold" />
                 </div>
                 <div class="flex gap-2">
                     <button @click="saveList" class="px-6 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg font-medium transition-colors">Save List</button>
                     <button v-if="!isCreating" @click="confirmDelete(selectedList.ID)" class="px-4 py-2 bg-red-900/40 hover:bg-red-900/60 text-red-400 rounded-lg font-medium transition-colors">Delete</button>
                 </div>
             </div>

             <!-- Split View -->
             <div class="flex-1 flex gap-4 min-h-0">
                 <!-- Available Channels -->
                 <div class="flex-1 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden">
                      <div class="p-3 bg-slate-900/80 border-b border-slate-800 text-xs font-bold text-slate-500 uppercase">Available Channels</div>
                      <div class="flex-1 overflow-y-auto p-2 space-y-1">
                          <div v-for="ch in availableChannels" :key="ch.ID"
                               @click="selectedAvailableChId = ch.ID"
                               class="px-3 py-2 rounded-lg cursor-pointer text-sm transition-colors"
                               :class="selectedAvailableChId === ch.ID ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:bg-slate-800'">
                               {{ ch.Name }} <span class="opacity-50 text-xs ml-2">{{ ch.RxFrequency }}</span>
                          </div>
                      </div>
                 </div>

                 <!-- Controls -->
                 <div class="flex flex-col justify-center gap-2">
                     <button @click="addToList" class="p-2 bg-slate-800 hover:bg-slate-700 rounded text-slate-300 transition-colors" title="Add">
                         <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
                     </button>
                     <button @click="removeFromList" class="p-2 bg-slate-800 hover:bg-slate-700 rounded text-slate-300 transition-colors" title="Remove">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"></polyline></svg>
                     </button>
                 </div>

                 <!-- Members -->
                 <div class="flex-1 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden">
                      <div class="p-3 bg-slate-900/80 border-b border-slate-800 text-xs font-bold text-slate-500 uppercase">List Members</div>
                      <div class="flex-1 overflow-y-auto p-2 space-y-1">
                          <div v-for="(ch, idx) in selectedList.Channels" :key="idx"
                               @click="selectedMemberIndex = idx"
                               class="px-3 py-2 rounded-lg cursor-pointer text-sm transition-colors flex justify-between"
                               :class="selectedMemberIndex === idx ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:bg-slate-800'">
                               <span>{{ idx + 1 }}. {{ ch.Name }}</span>
                               <span class="opacity-50 font-mono">{{ ch.RxFrequency }}</span>
                          </div>
                      </div>
                 </div>

                 <!-- Reorder Controls -->
                  <div class="flex flex-col justify-center gap-2">
                     <button @click="moveUp" class="p-2 bg-slate-800 hover:bg-slate-700 rounded text-slate-300 transition-colors" title="Move Up">
                         <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"></line><polyline points="5 12 12 5 19 12"></polyline></svg>
                     </button>
                     <button @click="moveDown" class="p-2 bg-slate-800 hover:bg-slate-700 rounded text-slate-300 transition-colors" title="Move Down">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><polyline points="19 12 12 19 5 12"></polyline></svg>
                     </button>
                 </div>
             </div>
        </div>
        <div v-else class="flex-1 flex items-center justify-center text-slate-500">
            Select a list to edit or create a new one.
        </div>
    </div>

    <!-- Confirm Delete Modal -->
    <div v-if="showDeleteConfirm" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
        <div class="bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-2xl max-w-sm w-full">
            <h3 class="text-lg font-bold text-slate-100 mb-2">Delete Scan List?</h3>
            <p class="text-slate-400 mb-6 text-sm">Are you sure you want to delete this scan list? This action cannot be undone.</p>
            <div class="flex justify-end gap-3">
                <button @click="showDeleteConfirm = false" class="px-4 py-2 rounded-lg bg-slate-800 text-slate-300 hover:bg-slate-700 font-medium transition-colors">Cancel</button>
                <button @click="deleteList" class="px-4 py-2 rounded-lg bg-red-600 text-white hover:bg-red-500 font-medium transition-colors">Delete</button>
            </div>
        </div>
    </div>
  </div>
</template>
