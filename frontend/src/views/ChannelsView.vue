<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import { useCodeplugStore, type Channel } from '../stores/codeplug'
import Draggable from 'vuedraggable'

const store = useCodeplugStore()
const searchQuery = ref('')
const sortColumn = ref<keyof Channel | null>(null) // Null = default ID sort (drag enabled)
const sortDirection = ref<'asc'|'desc'>('asc')
const editingChannelId = ref<number | null>(null) // Which channel row is being edited
const showModal = ref(false)
const modalChannel = ref<Partial<Channel> | null>(null) // Partial for bulk edit support
const isBulkMode = ref(false)
const selectedChannelIds = ref<Set<number>>(new Set())
// Notification state
const notification = ref<{message: string, type: 'success' | 'error'} | null>(null)

onMounted(() => {
    store.fetchChannels()
    store.fetchTalkgroups() // Needed for contact dropdown
    store.fetchScanLists() // Needed for scan list dropdown
  })

const filteredChannels = computed({
    get() {
        // Basic filtering
        let res = store.channels.filter(ch => 
            ch.Name.toLowerCase().includes(searchQuery.value.toLowerCase())
        )

        // Sorting
        // If sortColumn is set, we sort.
        // If sortColumn is NULL (default), we assume channels are naturally sorted by ID
        // Draggable needs the list to be mutable or update the store.
        // Reordering is only allowed when NO searchQuery AND NO custom sort.
        if (sortColumn.value) {
            res.sort((a, b) => {
                let valA = a[sortColumn.value!]
                let valB = b[sortColumn.value!]
                if (typeof valA === 'string') valA = valA.toLowerCase()
                if (typeof valB === 'string') valB = valB.toLowerCase()
                // Handle null/undefined (treat as empty string or 0)
                if (valA === null || valA === undefined) valA = ''
                if (valB === null || valB === undefined) valB = ''

                if (valA < valB) return sortDirection.value === 'asc' ? -1 : 1
                if (valA > valB) return sortDirection.value === 'asc' ? 1 : -1
                return 0
            })
        }
        return res
    },
    // We don't really use the setter for reordering directly via v-model on filteredChannels
    // because filteredChannels is computed. We use @change/v-model on a simpler ref or handle @end.
    // However, VueDraggable requires v-model/list binding.
    // Binding to `store.channels` directly is unsafe if we filter it.
    // We will use the `:list` prop if readonly, but draggable works best with v-model.
    set(val) {
        // Not used with simpler read-only list for dragging only when raw
    }
})

// Drag enabled condition
const canDrag = computed(() => {
    return searchQuery.value === '' && (sortColumn.value === null || sortColumn.value === 'ID') && sortDirection.value === 'asc'
})

// Handle reorder
const onDragEnd = async (evt: any) => {
    if (evt.oldIndex === evt.newIndex) return

    // We need the new order of IDs.
    // The `filteredChannels` view might not reflect the store immediately if we don't mutate.
    // vuedraggable modifies the array passed to `list` or `modelValue`.
    // Since filteredChannels is a computed getter, we can't bind v-model nicely without complications.
    // Strategy: Use a local mutable copy for the list? 
    // Or just bind to `store.channels` and ensuring we modify store state?
    // Let's use `list` prop which points to `store.channels` (since we verified canDrag implies no filter).
    
    // Actually, store.channels is readonly ref from store? No, it's a ref. But mutating it directly doesn't Trigger Backend Save.
    // We need to trigger backend save.
    
    // We should construct the new ID list from the DOM or the evt?
    // vuedraggable updates the bound array in place.
    
    // Let's create a computed writable wrapper for store.channels
}

// Safer approach for Draggable + Pinia:
// Bind v-model to a computed that:
// get: returns store.channels
// set: calls store which updates channels (but doesn't save yet?)
// Wait, we want to SAVE heavily on drop.
// We can use `@end` event to detect drop, grab the new list of IDs, and send to backend.

const dragOptions = {
    animation: 200,
    disabled: !canDrag.value,
    ghostClass: "ghost",
    handle: ".drag-handle"
}

const onReorder = async (evt: any) => {
    // The `store.channels` array has been updated by v-model?
    // Only if we provide a writable computed.
    // Let's look at `store.channels`
    
    // Implementation:
    // We bind v-model="dragList"
    // dragList is a ref synced with store.channels.
}
const dragList = ref<Channel[]>([])
watch(() => store.channels, (newVal) => {
    dragList.value = [...newVal]
}, { deep: true, immediate: true })

const updateOrder = async () => {
    if (!canDrag.value) return
    // dragList is now in new order.
    const ids = dragList.value.map(c => c.ID)
    try {
        await store.reorderChannels(ids)
        // store.channels is updated by fetchChannels inside reorderChannels
        showNotification("Channels reordered")
    } catch (e: any) {
        showNotification(e.message || "Reorder failed", "error")
        // Revert by re-fetching (or just resetting dragList)
        // Since store.channels wasn't updated (if fetch didn't happen or returned old), 
        // resetting dragList to store.channels should revert UI.
        dragList.value = [...store.channels]
    }
}

// ... existing logic ...


const handleSort = (column: keyof Channel) => {
    if (sortColumn.value === column) {
        sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
    } else {
        sortColumn.value = column
        sortDirection.value = 'asc'
    }
}

// Selection Logic
const toggleSelection = (id: number) => {
    if (selectedChannelIds.value.has(id)) {
        selectedChannelIds.value.delete(id)
    } else {
        selectedChannelIds.value.add(id)
    }
}

const toggleSelectAll = () => {
    if (selectedChannelIds.value.size === filteredChannels.value.length) {
        selectedChannelIds.value.clear()
    } else {
        filteredChannels.value.forEach(ch => selectedChannelIds.value.add(ch.ID))
    }
}

const isAllSelected = computed(() => {
    return filteredChannels.value.length > 0 && selectedChannelIds.value.size === filteredChannels.value.length
})

const isIndeterminate = computed(() => {
    return selectedChannelIds.value.size > 0 && selectedChannelIds.value.size < filteredChannels.value.length
})

// Inline Editing Logic
const startEditing = (id: number) => {
  editingChannelId.value = id
}

const stopEditing = () => {
    editingChannelId.value = null
}

const saveInline = async (ch: Channel) => {
    await store.saveChannel(ch)
    showNotification("Channel saved")
}

// Modal Logic for Advanced Fields
const openEditModal = (ch: Channel) => {
  isBulkMode.value = false
  modalChannel.value = JSON.parse(JSON.stringify(ch)) // Deep copy
  showModal.value = true
}

const openBulkEditModal = () => {
    if (selectedChannelIds.value.size === 0) return
    isBulkMode.value = true
    // Initialize with undefined values for bulk mode so we know what changed
    modalChannel.value = {
        // We only set defaults that are safe to be "No Change" if undefined
        // Actually, we can just start with an empty object and cast it
    } as any
    showModal.value = true
}

const openAddModal = () => {
    isBulkMode.value = false
    modalChannel.value = {
    ID: 0,
    Name: '',
    RxFrequency: 146.5200,
    TxFrequency: 146.5200,
    Mode: 'FM',
    Tone: '',
    Skip: false,
    SquelchType: 'None',
    RxTone: '',
    TxTone: '',
    RxDCS: '',
    TxDCS: '',
    Type: 'Analog',
    Protocol: 'FM',
    ColorCode: 1,
    TimeSlot: 1,
    ContactID: undefined,
    Power: 'High',
    Bandwidth: '12.5',
    ScanList: 'None',
    RxGroup: 'None',
    TxContact: 'None',
    TalkAround: false,
    WorkAlone: false,
    TxPermit: 'Always',
    RxSquelchMode: 'Normal',
    Notes: ''
  }
  showModal.value = true
}

const saveModal = async () => {
    if (!modalChannel.value) return

    if (isBulkMode.value) {
        if (!confirm(`Are you sure you want to update ${selectedChannelIds.value.size} channels?`)) return
        
        // Filter out undefined values from modalChannel
        const updates: Partial<Channel> = {}
        for (const [key, value] of Object.entries(modalChannel.value)) {
            if (value !== undefined && value !== null && value !== '') { // careful with empty string for Name etc, but for bulk operations usually we ignore empty unless explicit
               // For bulk, if user didn't touch it, it should be undefined in our logic (if we didn't init it)
               // But wait, v-model will initialize it? No, only if we provide initial value.
               updates[key as keyof Channel] = value
            }
        }
        
        // Use a better check: Only include keys that are actually present in modalChannel
        // Since we init as empty object, only v-modeled fields that were touched/set will be there? 
        // Vue v-model might require the property to exist to work reactively without warnings?
        // Let's rely on the fact that we can initialize with undefineds if needed, or check logic.
        
        await store.bulkUpdateChannels(modalChannel.value as Partial<Channel>, Array.from(selectedChannelIds.value))
        showModal.value = false
        modalChannel.value = null
        selectedChannelIds.value.clear()
        showNotification("Bulk update complete")
    } else {
        // Single Edit
        await store.saveChannel(modalChannel.value as Channel)
        // Do NOT close modal
        showNotification("Channel saved")
    }
}

const deleteChannel = async (id: number) => {
    if (confirm("Are you sure?")) {
        await store.deleteChannel(id)
        if (selectedChannelIds.value.has(id)) selectedChannelIds.value.delete(id)
    }
}

const deleteSelected = async () => {
    if (confirm(`Are you sure you want to delete ${selectedChannelIds.value.size} channels?`)) {
        // This might arguably be better as a bulk delete action in store, but iterating is fine for now
        for (const id of selectedChannelIds.value) {
            await store.deleteChannel(id)
        }
        selectedChannelIds.value.clear()
        showNotification("Channels deleted")
    }
}

const navigateChannel = (direction: number) => {
    if (!modalChannel.value || isBulkMode.value) return

    // Find current index in filtered list
    // Cast to Channel because in single mode ID is present
    const currentId = (modalChannel.value as Channel).ID
    const currentIndex = filteredChannels.value.findIndex(c => c.ID === currentId)
    if (currentIndex === -1) return

    const newIndex = currentIndex + direction
    if (newIndex >= 0 && newIndex < filteredChannels.value.length) {
        // Save current changes first? "Save & Stay" implies we save when clicking save. 
        // Navigating without saving usually discards changes or prompts. 
        // User asked "when we click save... notify but stay... that way we can use arrows". 
        // This implies save is manual. Navigation should probably just load the next one (discarding unsaved?).
        // Or should we auto-save? Let's stick to simple: Navigation just switches. User must click Save if they want to save.
        
        modalChannel.value = JSON.parse(JSON.stringify(filteredChannels.value[newIndex]))
    }
}

// Watch for Type changes to set default Bandwidth
watch(() => modalChannel.value?.Type, (newType) => {
    if (!modalChannel.value) return
    // Only apply default if we are adding a new channel or if logic allows. 
    // In bulk mode, we might not want to auto-set bandwidth unless user changed type explicitly?
    // This watcher fires on any change. 
    
    // Simplification for now: apply defaults.
    if (newType === 'Analog') {
        modalChannel.value.Bandwidth = '25'
    } else if (newType?.includes('Digital')) {
        modalChannel.value.Bandwidth = '12.5'
    }
})

// Inline input helpers
const updateRxFreq = (ch: Channel, val: string) => {
    ch.RxFrequency = parseFloat(parseFloat(val).toFixed(4))
}
const updateTxFreq = (ch: Channel, val: string) => {
    ch.TxFrequency = parseFloat(parseFloat(val).toFixed(4))
}

const showNotification = (msg: string, type: 'success'|'error' = 'success') => {
    notification.value = { message: msg, type }
    setTimeout(() => notification.value = null, 3000)
}

const channelTypes = ['Analog', 'Digital (DMR)', 'Digital (NXDN)', 'Digital (YSF)', 'Digital (D-Star)', 'Digital (P25)']
const powerLevels = ['High', 'Mid', 'Low', 'Turbo']
const bandwidths = ['12.5', '25']
const squelchTypes = ['None', 'Tone', 'TSQL', 'DCS']

</script>

<template>
  <div class="h-full flex flex-col relative">
    <!-- Notification Toast -->
    <div v-if="notification" 
         class="absolute bottom-4 right-4 z-50 px-4 py-2 rounded-lg shadow-lg text-sm font-medium transition-all transform duration-300"
         :class="notification.type === 'success' ? 'bg-emerald-500 text-white' : 'bg-red-500 text-white'"
    >
        {{ notification.message }}
    </div>

    <!-- Toolbar -->
    <div class="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-900/50 backdrop-blur-sm sticky top-0 z-10 gap-4">
      <div class="flex items-center gap-4 flex-1">
          <div class="relative w-96">
            <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-slate-500">
               <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
            </div>
            <input 
              v-model="searchQuery"
              type="text" 
              placeholder="Search channels..." 
              class="w-full pl-10 pr-4 py-2 bg-slate-950/50 border border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500/50 text-sm placeholder-slate-500 transition-all text-slate-200"
            >
          </div>
          
          <!-- Bulk Actions -->
          <div v-if="selectedChannelIds.size > 0" class="flex items-center gap-2 animate-in fade-in slide-in-from-left-4 duration-200">
              <span class="text-xs font-semibold text-slate-400 bg-slate-800 px-2 py-1 rounded-md">{{ selectedChannelIds.size }} Selected</span>
              <button @click="openBulkEditModal" class="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded-lg text-sm font-medium transition-all flex items-center gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                  Bulk Edit
              </button>
              <button @click="deleteSelected" class="px-3 py-1.5 bg-red-900/30 hover:bg-red-900/50 text-red-400 rounded-lg text-sm font-medium transition-all flex items-center gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                  Delete
              </button>
          </div>
      </div>

      <button @click="openAddModal" class="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-sm font-medium shadow-lg shadow-indigo-500/20 transition-all flex-shrink-0">
          Add Channel
      </button>
    </div>

    <!-- Table Container -->
    <div class="flex-1 overflow-auto bg-slate-900">
      <table class="text-left border-collapse w-max min-w-full">
        <thead class="sticky top-0 bg-slate-900 z-30 shadow-sm border-b border-slate-700">
          <tr class="text-xs uppercase tracking-wider text-slate-500 font-semibold h-10">
            <!-- Sticky Utility Columns -->
            <th class="px-3 py-2 bg-slate-900 sticky left-0 z-40 w-10 text-center border-r border-slate-800">
                 <input type="checkbox" 
                        :checked="isAllSelected" 
                        :indeterminate="isIndeterminate"
                        @change="toggleSelectAll"
                        class="rounded bg-slate-950 border-slate-700 text-indigo-600 focus:ring-indigo-500/50 cursor-pointer" />
            </th>
            <th class="px-3 py-2 bg-slate-900 sticky left-10 z-40 w-24 text-center border-r border-slate-800">Actions</th>
            <th @click="handleSort('ID')" class="px-3 py-2 bg-slate-900 sticky left-[136px] z-40 w-16 text-right border-r border-slate-800 cursor-pointer hover:text-white">
                <div class="flex items-center justify-end gap-1">
                    ID
                    <span v-if="sortColumn === 'ID'" class="text-indigo-500">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
                </div>
            </th>

            <!-- Sortable Data Columns -->
            <th @click="handleSort('Name')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50 hover:text-white transition-colors group">
                <div class="flex items-center gap-1">
                    Name
                    <span v-if="sortColumn === 'Name'" class="text-indigo-500">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
                </div>
            </th>
            <th @click="handleSort('RxFrequency')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50 hover:text-white transition-colors">
                <div class="flex items-center gap-1">
                    Rx Freq
                    <span v-if="sortColumn === 'RxFrequency'" class="text-indigo-500">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
                </div>
            </th>
            <th @click="handleSort('TxFrequency')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50 hover:text-white transition-colors">
                 <div class="flex items-center gap-1">
                    Tx Freq
                    <span v-if="sortColumn === 'TxFrequency'" class="text-indigo-500">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
                </div>
            </th>
            <th @click="handleSort('Type')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50 hover:text-white transition-colors">
                 <div class="flex items-center gap-1">
                    Type
                    <span v-if="sortColumn === 'Type'" class="text-indigo-500">{{ sortDirection === 'asc' ? '↑' : '↓' }}</span>
                </div>
            </th>
            <th @click="handleSort('Power')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50 hover:text-white transition-colors">Power</th>
            <th @click="handleSort('Bandwidth')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50 hover:text-white transition-colors">BW</th>
            
            <th class="px-4 py-2 bg-slate-900/50 text-slate-400 border-l border-slate-800/50">Squelch</th>
            <th @click="handleSort('RxTone')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50">Rx Tone</th>
            <th @click="handleSort('TxTone')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50">Tx Tone</th>
            <th @click="handleSort('RxDCS')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50">Rx DCS</th>
            <th @click="handleSort('TxDCS')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50">Tx DCS</th>

            <th @click="handleSort('ColorCode')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50 border-l border-slate-800/50">CC</th>
            <th @click="handleSort('TimeSlot')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50">Slot</th>
            <th @click="handleSort('ContactID')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50">Tx Contact</th>
            <th @click="handleSort('RxGroup')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50">Rx Group</th>

            <th @click="handleSort('ScanList')" class="px-4 py-2 cursor-pointer hover:bg-slate-800/50 border-l border-slate-800/50">Scan List</th>
            <th class="px-4 py-2 border-l border-slate-800/50">Flags</th>
            <th class="px-4 py-2">Notes</th>
          </tr>
        </thead>
        <Draggable v-model="dragList" 
                   tag="tbody" 
                   item-key="ID"
                   handle=".drag-handle"
                   :animation="200"
                   :disabled="!canDrag"
                   class="divide-y divide-slate-800/50"
                   @end="updateOrder">
          <template #item="{ element: ch }">
           <tr class="group hover:bg-slate-800/30 transition-colors text-sm"
              :class="{
                  'bg-slate-800/80': editingChannelId === ch.ID,
                  'bg-indigo-900/10': selectedChannelIds.has(ch.ID)
              }">
            
            <!-- Drag Handle -->
            <td class="px-1 py-2 text-center sticky left-0 z-40 bg-slate-900 border-r border-slate-800 group-hover:bg-slate-900/90 transition-colors cursor-move drag-handle opacity-50 hover:opacity-100"
                :class="{'cursor-not-allowed opacity-20': !canDrag}">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" class="mx-auto text-slate-500"><path stroke="currentColor" stroke-width="2" stroke-linecap="round" d="M9 5h.01M15 5h.01M9 12h.01M15 12h.01M9 19h.01M15 19h.01"/></svg>
            </td>

            <!-- Sticky Checkbox (Shifted left offset) -->
            <td class="px-3 py-2 text-center sticky left-8 z-20 bg-slate-900 border-r border-slate-800 group-hover:bg-slate-900/90 transition-colors">
                <input type="checkbox" 
                       :checked="selectedChannelIds.has(ch.ID)"
                       @change="toggleSelection(ch.ID)"
                       class="rounded bg-slate-950 border-slate-700 text-indigo-600 focus:ring-indigo-500/50 cursor-pointer" />
            </td>

            <!-- Sticky Actions (Shifted) -->
             <td class="px-3 py-2 text-center sticky left-[72px] z-20 bg-slate-900 border-r border-slate-800 group-hover:bg-slate-900/90 transition-colors whitespace-nowrap">
               <div class="flex items-center justify-center gap-1">
                 <button v-if="editingChannelId !== ch.ID" @click="startEditing(ch.ID)" class="p-1 hover:bg-slate-700 rounded text-slate-400 hover:text-indigo-400 transition-colors" title="Quick Edit">
                     <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"></path><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"></path></svg>
                 </button>
                 <button v-else @click="stopEditing()" class="p-1 hover:bg-slate-700 rounded text-green-400 hover:text-green-300 transition-colors" title="Finish">
                     <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
                 </button>
                  <button @click="openEditModal(ch)" class="p-1 hover:bg-slate-700 rounded text-slate-400 hover:text-white transition-colors" title="Full Edit">
                   <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                 </button>
                  <button @click="deleteChannel(ch.ID)" class="p-1 hover:bg-red-900/30 rounded text-slate-400 hover:text-red-400 transition-colors" title="Delete">
                   <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                 </button>
               </div>
            </td>

            <!-- Sticky ID -->
            <td class="px-3 py-2 text-right text-slate-500 font-mono text-xs sticky left-[168px] z-20 bg-slate-900 border-r border-slate-800 group-hover:bg-slate-900/90 transition-colors">
                {{ ch.ID }}
            </td>
            
            <!-- Name -->
            <td class="px-4 py-2 min-w-[200px]">
                <input v-if="editingChannelId === ch.ID" 
                       v-model="ch.Name" 
                       @blur="saveInline(ch)"
                       class="bg-slate-950 border border-indigo-500/50 rounded px-2 py-1 text-sm text-white w-full focus:outline-none" />
                <span v-else @dblclick="startEditing(ch.ID)" class="cursor-text block truncate">{{ ch.Name }}</span>
            </td>

             <!-- Rx Freq -->
            <td class="px-4 py-2 font-mono text-indigo-300 whitespace-nowrap">
                 <input v-if="editingChannelId === ch.ID" 
                       :value="ch.RxFrequency"
                       @input="(e) => updateRxFreq(ch, (e.target as HTMLInputElement).value)"
                       @blur="saveInline(ch)"
                       type="number" step="0.0001"
                       class="bg-slate-950 border border-indigo-500/50 rounded px-2 py-1 text-sm text-white w-24 focus:outline-none" />
                <span v-else @dblclick="startEditing(ch.ID)" class="cursor-text">{{ ch.RxFrequency.toFixed(4) }}</span>
            </td>

            <!-- Tx Freq -->
            <td class="px-4 py-2 font-mono text-slate-400 whitespace-nowrap">
                 <input v-if="editingChannelId === ch.ID" 
                       :value="ch.TxFrequency"
                       @input="(e) => updateTxFreq(ch, (e.target as HTMLInputElement).value)"
                       @blur="saveInline(ch)"
                        type="number" step="0.0001"
                       class="bg-slate-950 border border-indigo-500/50 rounded px-2 py-1 text-sm text-white w-24 focus:outline-none" />
                <span v-else @dblclick="startEditing(ch.ID)" class="cursor-text">{{ ch.TxFrequency.toFixed(4) }}</span>
            </td>

             <!-- Type -->
             <td class="px-4 py-2 whitespace-nowrap">
                 <select v-if="editingChannelId === ch.ID"
                         v-model="ch.Type"
                         @change="saveInline(ch)"
                         class="bg-slate-950 border border-indigo-500/50 rounded px-2 py-1 text-xs text-white focus:outline-none w-32">
                     <option v-for="t in channelTypes" :key="t" :value="t">{{ t }}</option>
                 </select>
                  <span v-else class="px-2 py-0.5 rounded-md text-[10px] uppercase font-bold tracking-wide border inline-block"
                    :class="{
                      'bg-emerald-500/10 text-emerald-400 border-emerald-500/20': ch.Type === 'Analog',
                      'bg-blue-500/10 text-blue-400 border-blue-500/20': ch.Type.includes('DMR'),
                      'bg-purple-500/10 text-purple-400 border-purple-500/20': ch.Type.includes('NXDN')
                    }"
                  >
                    {{ ch.Type.replace('Digital ', '') }}
                  </span>
             </td>

             <!-- Power / BW -->
             <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs">{{ ch.Power }}</td>
             <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs">{{ ch.Bandwidth }}K</td>

             <!-- Squelch -->
             <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs border-l border-slate-800/50">{{ ch.SquelchType === 'None' ? '-' : ch.SquelchType }}</td>
             <td class="px-4 py-2 whitespace-nowrap font-mono text-slate-400 text-xs">{{ ch.RxTone }}</td>
             <td class="px-4 py-2 whitespace-nowrap font-mono text-slate-400 text-xs">{{ ch.TxTone }}</td>
             <td class="px-4 py-2 whitespace-nowrap font-mono text-slate-400 text-xs">{{ ch.RxDCS }}</td>
             <td class="px-4 py-2 whitespace-nowrap font-mono text-slate-400 text-xs">{{ ch.TxDCS }}</td>

            <!-- Digital -->
             <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs border-l border-slate-800/50">
                 <span v-if="ch.Type.includes('Digital')">{{ ch.ColorCode }}</span>
                 <span v-else class="text-slate-700">-</span>
             </td>
             <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs">
                 <span v-if="ch.Type.includes('Digital')">{{ ch.TimeSlot }}</span>
                 <span v-else class="text-slate-700">-</span>
             </td>
             <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs">
                 <!-- Need lookup logic if ContactID is a number, for now assume store handles mapping or we show ID -->
                 <div v-if="ch.ContactID" class="flex items-center gap-1">
                      <!-- Simple lookup if we have the list, else show ID -->
                      {{ store.talkgroups.find(tg => tg.ID === ch.ContactID)?.Name || ch.ContactID }}
                 </div>
                 <span v-else class="text-slate-700">-</span>
             </td>
              <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs">
                 {{ ch.RxGroup }}
             </td>

             <!-- Other -->
             <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs border-l border-slate-800/50">{{ ch.ScanList }}</td>
             <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs border-l border-slate-800/50 flex gap-1">
                 <span v-if="ch.Skip" class="text-yellow-500" title="Skip">S</span>
                 <span v-if="ch.TalkAround" class="text-blue-500" title="Talk Around">TA</span>
                 <span v-if="ch.WorkAlone" class="text-red-500" title="Work Alone">WA</span>
             </td>
             <td class="px-4 py-2 whitespace-nowrap text-slate-400 text-xs max-w-[200px] truncate">{{ ch.Notes }}</td>

          </tr>
         </template>
        </Draggable>
        
        <!-- Empty State in separate tbody because Draggable takes over original tbody -->
         <tbody v-if="filteredChannels.length === 0">
           <tr>
                <td colspan="21" class="px-6 py-12 text-center text-slate-500">
                  No channels found (or clear search to reorder).
                </td>
           </tr>
        </tbody>
      </table>
    </div>
    
    <!-- Comprehensive Modal -->
    <div v-if="showModal && modalChannel" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
        <div class="bg-slate-900 border border-slate-700 rounded-2xl p-6 w-full max-w-4xl shadow-2xl overflow-y-auto max-h-[90vh]">
            <h2 class="text-xl font-bold mb-4 border-b border-slate-800 pb-2 flex items-center gap-2">
                 <span v-if="isBulkMode" class="text-indigo-400">Bulk Edit ({{ selectedChannelIds.size }} Channels)</span>
                 <span v-else>{{ modalChannel.ID === 0 ? 'Add Channel' : 'Edit Channel' }}</span>
            </h2>
            
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                
                <!-- Basic Info -->
                <div class="col-span-full mb-2">
                    <h3 class="text-sm font-semibold text-indigo-400 mb-2">General</h3>
                </div>

                <!-- Fields hidden in bulk mode -->
                <template v-if="!isBulkMode">
                    <div class="col-span-2">
                        <label class="block text-xs font-medium text-slate-400 mb-1">Name</label>
                        <input v-model="modalChannel.Name" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                    </div>
                     <div>
                        <label class="block text-xs font-medium text-slate-400 mb-1">Rx Freq</label>
                        <input v-model.number="modalChannel.RxFrequency" type="number" step="0.0001" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                    </div>
                     <div>
                        <label class="block text-xs font-medium text-slate-400 mb-1">Tx Freq</label>
                        <input v-model.number="modalChannel.TxFrequency" type="number" step="0.0001" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                    </div>
                </template>

                 <div>
                    <label class="block text-xs font-medium text-slate-400 mb-1">Type <span v-if="isBulkMode" class="text-xs text-slate-600">(leave to keep)</span></label>
                     <select v-model="modalChannel.Type" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-if="isBulkMode" :value="undefined">No Change</option>
                        <option v-for="t in channelTypes" :key="t" :value="t">{{ t }}</option>
                    </select>
                </div>
                <div>
                     <label class="block text-xs font-medium text-slate-400 mb-1">Power</label>
                     <select v-model="modalChannel.Power" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-if="isBulkMode" :value="undefined">No Change</option>
                        <option v-for="p in powerLevels" :key="p" :value="p">{{ p }}</option>
                    </select>
                </div>
                 <div>
                     <label class="block text-xs font-medium text-slate-400 mb-1">Bandwidth</label>
                     <select v-model="modalChannel.Bandwidth" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-if="isBulkMode" :value="undefined">No Change</option>
                        <option v-for="b in bandwidths" :key="b" :value="b">{{ b }}K</option>
                    </select>
                </div>
                 <div>
                     <label class="block text-xs font-medium text-slate-400 mb-1">Scan List</label>
                     <select v-model="modalChannel.ScanList" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-if="isBulkMode" :value="undefined">No Change</option>
                        <option value="None">None</option>
                        <option v-for="sl in store.scanlists" :key="sl.ID" :value="sl.Name">{{ sl.Name }}</option>
                    </select>
                </div>
                
                <!-- Analog Specific -->
                 <div class="col-span-full mt-2 mb-2 pt-2 border-t border-slate-800">
                    <h3 class="text-sm font-semibold text-emerald-400 mb-2">Analog / Squelch</h3>
                </div>

                <div>
                     <label class="block text-xs font-medium text-slate-400 mb-1">Squelch Mode</label>
                     <select v-model="modalChannel.SquelchType" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-if="isBulkMode" :value="undefined">No Change</option>
                        <option v-for="s in squelchTypes" :key="s" :value="s">{{ s }}</option>
                    </select>
                </div>
                 <!-- In Bulk Mode, show these inputs only if SquelchType is NOT explicitly incompatible OR if we are in bulk mode we might want to allow setting them regardless? 
                      Allow setting them regardless is safest, user knows what they are doing. -->
                 <div v-if="!isBulkMode ? (modalChannel.SquelchType === 'Tone' || modalChannel.SquelchType === 'TSQL') : true">
                     <!-- For bulk mode we show it always? Or maybe only if user selects appropriate SquelchType? 
                          Let's show it always in bulk mode so they can set Tones blindly if they want. -->
                    <label class="block text-xs font-medium text-slate-400 mb-1">Rx Tone</label>
                    <input v-model="modalChannel.RxTone" placeholder="e.g. 88.5" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                <div v-if="!isBulkMode ? (modalChannel.SquelchType === 'Tone' || modalChannel.SquelchType === 'TSQL') : true">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Tx Tone</label>
                    <input v-model="modalChannel.TxTone" placeholder="e.g. 88.5" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                  <div v-if="!isBulkMode ? (modalChannel.SquelchType === 'DCS') : true">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Rx DCS</label>
                    <input v-model="modalChannel.RxDCS" placeholder="e.g. D023N" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                <div v-if="!isBulkMode ? (modalChannel.SquelchType === 'DCS') : true">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Tx DCS</label>
                    <input v-model="modalChannel.TxDCS" placeholder="e.g. D023N" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>


                <!-- Digital Specific -->
                <div v-if="!isBulkMode ? modalChannel?.Type?.includes('Digital') : true" class="col-span-full mt-2 mb-2 pt-2 border-t border-slate-800">
                    <h3 class="text-sm font-semibold text-blue-400 mb-2">Digital (DMR/NXDN)</h3>
                </div>

                 <div v-if="!isBulkMode ? modalChannel?.Type?.includes('Digital') : true">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Color Code</label>
                    <input v-model.number="modalChannel.ColorCode" type="number" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                 <div v-if="!isBulkMode ? modalChannel?.Type?.includes('Digital') : true">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Time Slot</label>
                     <select v-model.number="modalChannel.TimeSlot" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-if="isBulkMode" :value="undefined">No Change</option>
                        <option :value="1">Slot 1</option>
                        <option :value="2">Slot 2</option>
                    </select>
                </div>
                 <div v-if="!isBulkMode ? modalChannel?.Type?.includes('Digital') : true">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Rx Group List</label>
                    <input v-model="modalChannel.RxGroup" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                 <div v-if="!isBulkMode ? modalChannel?.Type?.includes('Digital') : true" class="col-span-2">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Tx Contact (Talkgroup)</label>
                    <select v-model="modalChannel.ContactID" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-if="isBulkMode" :value="undefined">No Change</option>
                        <option :value="undefined">None</option>
                        <option v-for="tg in store.talkgroups" :key="tg.ID" :value="tg.ID">
                            {{ tg.Name }} ({{ tg.Type }})
                        </option>
                    </select>
                </div>
                
                 <!-- Flags -->
                 <div class="col-span-full mt-2 mb-2 pt-2 border-t border-slate-800">
                    <h3 class="text-sm font-semibold text-slate-400 mb-2">Flags</h3>
                </div>

                 <div class="col-span-full flex flex-wrap gap-4">
                     <label class="flex items-center gap-2 text-slate-300">
                         <input type="checkbox" v-model="modalChannel.Skip" class="rounded bg-slate-950 border-slate-700 text-indigo-600 focus:ring-indigo-500/50">
                         Skip Scanning
                     </label>
                      <label class="flex items-center gap-2 text-slate-300">
                         <input type="checkbox" v-model="modalChannel.TalkAround" class="rounded bg-slate-950 border-slate-700 text-indigo-600 focus:ring-indigo-500/50">
                         Talk Around
                     </label>
                      <label class="flex items-center gap-2 text-slate-300">
                         <input type="checkbox" v-model="modalChannel.WorkAlone" class="rounded bg-slate-950 border-slate-700 text-indigo-600 focus:ring-indigo-500/50">
                         Work Alone
                     </label>
                 </div>
            </div>

            <div class="mt-6 flex justify-between gap-3 pt-4 border-t border-slate-800">
                <div class="flex gap-2">
                     <button v-if="!isBulkMode" @click="navigateChannel(-1)" class="px-3 py-2 rounded-lg bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white" title="Previous Channel">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>
                    </button>
                    <button v-if="!isBulkMode" @click="navigateChannel(1)" class="px-3 py-2 rounded-lg bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white" title="Next Channel">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
                    </button>
                </div>
                <div class="flex gap-3">
                    <button @click="showModal = false" class="px-4 py-2 rounded-lg bg-slate-800 text-slate-300 hover:bg-slate-700">Cancel</button>
                    <button @click="saveModal" class="px-4 py-2 rounded-lg bg-indigo-600 text-white hover:bg-indigo-500">{{ isBulkMode ? 'Update All' : 'Save' }}</button>
                </div>
            </div>
        </div>
    </div>
  </div>
</template>
