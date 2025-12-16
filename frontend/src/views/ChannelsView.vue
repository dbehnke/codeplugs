<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useCodeplugStore, type Channel } from '../stores/codeplug'

const store = useCodeplugStore()
const searchQuery = ref('')
const editingChannelId = ref<number | null>(null) // Which channel row is being edited
const showModal = ref(false)
const modalChannel = ref<Channel | null>(null)

onMounted(() => {
    store.fetchChannels()
    store.fetchTalkgroups() // Needed for contact dropdown
    store.fetchScanLists() // Needed for scan list dropdown
  })

const filteredChannels = computed(() => {
  return store.channels.filter(ch => 
    ch.Name.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
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
    // Optional: Toast notification
}

// Modal Logic for Advanced Fields
const openEditModal = (ch: Channel) => {
  modalChannel.value = JSON.parse(JSON.stringify(ch)) // Deep copy
  showModal.value = true
}

const openAddModal = () => {
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
    if (modalChannel.value) {
        // Map TxContact name to ID if selected from dropdown?
        // For now, we bind ContactID to the select value
        await store.saveChannel(modalChannel.value)
        showModal.value = false
        modalChannel.value = null
    }
}

const deleteChannel = async (id: number) => {
    if (confirm("Are you sure?")) {
        await store.deleteChannel(id)
    }
}

const navigateChannel = (direction: number) => {
    if (!modalChannel.value) return

    // Find current index in filtered list
    const currentIndex = filteredChannels.value.findIndex(c => c.ID === modalChannel.value?.ID)
    if (currentIndex === -1) return

    const newIndex = currentIndex + direction
    if (newIndex >= 0 && newIndex < filteredChannels.value.length) {
        // Switch to new channel (Deep Copy)
        modalChannel.value = JSON.parse(JSON.stringify(filteredChannels.value[newIndex]))
    }
}

// Watch for Type changes to set default Bandwidth
import { watch } from 'vue'
watch(() => modalChannel.value?.Type, (newType) => {
    if (!modalChannel.value) return
    // Only apply default if the user hasn't explicitly set a non-default (optional enhancement, but for now simple overwrite is safer for consistency)
    // Or just overwrite on change.
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

const channelTypes = ['Analog', 'Digital (DMR)', 'Digital (NXDN)', 'Digital (YSF)', 'Digital (D-Star)', 'Digital (P25)']
const powerLevels = ['High', 'Mid', 'Low', 'Turbo']
const bandwidths = ['12.5', '25']
const squelchTypes = ['None', 'Tone', 'TSQL', 'DCS']

</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Toolbar -->
    <div class="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-900/50 backdrop-blur-sm sticky top-0 z-10">
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
      <button @click="openAddModal" class="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-sm font-medium shadow-lg shadow-indigo-500/20 transition-all">
          Add Channel
      </button>
    </div>

    <!-- Table Container -->
    <div class="flex-1 overflow-auto">
      <table class="w-full text-left border-collapse">
        <thead class="sticky top-0 bg-slate-900 z-10 shadow-sm">
          <tr class="text-xs uppercase tracking-wider text-slate-500 font-semibold">
            <th class="px-6 py-4 bg-slate-900 sticky left-0 z-20 w-16">ID</th>
            <th class="px-6 py-4 bg-slate-900">Name</th>
            <th class="px-6 py-4 bg-slate-900">Rx Freq</th>
            <th class="px-6 py-4 bg-slate-900">Tx Freq</th>
             <th class="px-6 py-4 bg-slate-900">Type</th>
            <th class="px-6 py-4 bg-slate-900">Color Code</th>
            <th class="px-6 py-4 bg-slate-900 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/50">
          <tr v-for="ch in filteredChannels" :key="ch.ID" 
              class="group hover:bg-slate-800/30 transition-colors"
              :class="{'bg-slate-800/50': editingChannelId === ch.ID}">
            
            <td class="px-6 py-3 text-slate-500 font-mono text-xs sticky left-0 bg-slate-900/0 group-hover:bg-slate-800/0">{{ ch.ID }}</td>
            
            <!-- Name -->
            <td class="px-6 py-3">
                <input v-if="editingChannelId === ch.ID" 
                       v-model="ch.Name" 
                       @blur="saveInline(ch)"
                       class="bg-slate-950 border border-indigo-500/50 rounded px-2 py-1 text-sm text-white w-full focus:outline-none" />
                <span v-else @dblclick="startEditing(ch.ID)" class="cursor-text">{{ ch.Name }}</span>
            </td>

             <!-- Rx Freq -->
            <td class="px-6 py-3 font-mono text-indigo-300">
                 <input v-if="editingChannelId === ch.ID" 
                       :value="ch.RxFrequency"
                       @input="(e) => updateRxFreq(ch, (e.target as HTMLInputElement).value)"
                       @blur="saveInline(ch)"
                       type="number" step="0.0001"
                       class="bg-slate-950 border border-indigo-500/50 rounded px-2 py-1 text-sm text-white w-28 focus:outline-none" />
                <span v-else @dblclick="startEditing(ch.ID)" class="cursor-text">{{ ch.RxFrequency.toFixed(4) }}</span>
            </td>

            <!-- Tx Freq -->
            <td class="px-6 py-3 font-mono text-slate-400">
                 <input v-if="editingChannelId === ch.ID" 
                       :value="ch.TxFrequency"
                       @input="(e) => updateTxFreq(ch, (e.target as HTMLInputElement).value)"
                       @blur="saveInline(ch)"
                        type="number" step="0.0001"
                       class="bg-slate-950 border border-indigo-500/50 rounded px-2 py-1 text-sm text-white w-28 focus:outline-none" />
                <span v-else @dblclick="startEditing(ch.ID)" class="cursor-text">{{ ch.TxFrequency.toFixed(4) }}</span>
            </td>

             <!-- Type -->
             <td class="px-6 py-3">
                 <select v-if="editingChannelId === ch.ID"
                         v-model="ch.Type"
                         @change="saveInline(ch)"
                         class="bg-slate-950 border border-indigo-500/50 rounded px-2 py-1 text-xs text-white focus:outline-none">
                     <option v-for="t in channelTypes" :key="t" :value="t">{{ t }}</option>
                 </select>
                  <span v-else class="px-2 py-1 rounded-md text-xs font-medium border"
                    :class="{
                      'bg-emerald-500/10 text-emerald-400 border-emerald-500/20': ch.Type === 'Analog',
                      'bg-blue-500/10 text-blue-400 border-blue-500/20': ch.Type.includes('DMR'),
                      'bg-purple-500/10 text-purple-400 border-purple-500/20': ch.Type.includes('NXDN')
                    }"
                  >
                    {{ ch.Type }}
                  </span>
             </td>

            <!-- Color Code (Only for Digital) -->
            <td class="px-6 py-3">
                <div v-if="ch.Type.includes('Digital')">
                    <input v-if="editingChannelId === ch.ID" 
                           v-model.number="ch.ColorCode"
                           @blur="saveInline(ch)"
                           type="number" min="0" max="15"
                           class="bg-slate-950 border border-indigo-500/50 rounded px-2 py-1 text-sm text-white w-16 focus:outline-none" />
                    <span v-else @dblclick="startEditing(ch.ID)" class="cursor-text">{{ ch.ColorCode }}</span>
                </div>
                <span v-else class="text-slate-600">-</span>
            </td>

            <td class="px-6 py-3 text-right whitespace-nowrap">
              <button @click="openEditModal(ch)" class="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors" title="Full Edit">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
              </button>
               <button v-if="editingChannelId !== ch.ID" @click="startEditing(ch.ID)" class="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-indigo-400 transition-colors" title="Quick Edit">
                 <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"></path><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"></path></svg>
              </button>
               <button v-else @click="stopEditing()" class="p-2 hover:bg-slate-700 rounded-lg text-green-400 hover:text-green-300 transition-colors" title="Finish">
                 <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
              </button>
              <button @click="deleteChannel(ch.ID)" class="p-2 hover:bg-red-900/30 rounded-lg text-slate-400 hover:text-red-400 transition-colors ml-1" title="Delete">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
              </button>
            </td>
          </tr>
           <tr v-if="filteredChannels.length === 0">
                <td colspan="7" class="px-6 py-12 text-center text-slate-500">
                  No channels found.
                </td>
           </tr>
        </tbody>
      </table>
    </div>
    
    <!-- Comprehensive Modal -->
    <div v-if="showModal && modalChannel" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
        <div class="bg-slate-900 border border-slate-700 rounded-2xl p-6 w-full max-w-4xl shadow-2xl overflow-y-auto max-h-[90vh]">
            <h2 class="text-xl font-bold mb-4 border-b border-slate-800 pb-2">{{ modalChannel.ID === 0 ? 'Add Channel' : 'Edit Channel' }}</h2>
            
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                
                <!-- Basic Info -->
                <div class="col-span-full mb-2">
                    <h3 class="text-sm font-semibold text-indigo-400 mb-2">General</h3>
                </div>

                <div class="col-span-2">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Name</label>
                    <input v-model="modalChannel.Name" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                 <div>
                    <label class="block text-xs font-medium text-slate-400 mb-1">Type</label>
                     <select v-model="modalChannel.Type" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-for="t in channelTypes" :key="t" :value="t">{{ t }}</option>
                    </select>
                </div>

                 <div>
                    <label class="block text-xs font-medium text-slate-400 mb-1">Rx Freq</label>
                    <input v-model.number="modalChannel.RxFrequency" type="number" step="0.0001" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                 <div>
                    <label class="block text-xs font-medium text-slate-400 mb-1">Tx Freq</label>
                    <input v-model.number="modalChannel.TxFrequency" type="number" step="0.0001" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                <div>
                     <label class="block text-xs font-medium text-slate-400 mb-1">Power</label>
                     <select v-model="modalChannel.Power" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-for="p in powerLevels" :key="p" :value="p">{{ p }}</option>
                    </select>
                </div>
                 <div>
                     <label class="block text-xs font-medium text-slate-400 mb-1">Bandwidth</label>
                     <select v-model="modalChannel.Bandwidth" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option v-for="b in bandwidths" :key="b" :value="b">{{ b }}K</option>
                    </select>
                </div>
                 <div>
                     <label class="block text-xs font-medium text-slate-400 mb-1">Scan List</label>
                     <select v-model="modalChannel.ScanList" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
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
                        <option v-for="s in squelchTypes" :key="s" :value="s">{{ s }}</option>
                    </select>
                </div>
                 <div v-if="modalChannel.SquelchType === 'Tone' || modalChannel.SquelchType === 'TSQL'">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Rx Tone</label>
                    <input v-model="modalChannel.RxTone" placeholder="e.g. 88.5" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                <div v-if="modalChannel.SquelchType === 'Tone' || modalChannel.SquelchType === 'TSQL'">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Tx Tone</label>
                    <input v-model="modalChannel.TxTone" placeholder="e.g. 88.5" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                  <div v-if="modalChannel.SquelchType === 'DCS'">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Rx DCS</label>
                    <input v-model="modalChannel.RxDCS" placeholder="e.g. D023N" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                <div v-if="modalChannel.SquelchType === 'DCS'">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Tx DCS</label>
                    <input v-model="modalChannel.TxDCS" placeholder="e.g. D023N" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>


                <!-- Digital Specific -->
                <div v-if="modalChannel.Type.includes('Digital')" class="col-span-full mt-2 mb-2 pt-2 border-t border-slate-800">
                    <h3 class="text-sm font-semibold text-blue-400 mb-2">Digital (DMR/NXDN)</h3>
                </div>

                 <div v-if="modalChannel.Type.includes('Digital')">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Color Code</label>
                    <input v-model.number="modalChannel.ColorCode" type="number" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                 <div v-if="modalChannel.Type.includes('Digital')">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Time Slot</label>
                     <select v-model.number="modalChannel.TimeSlot" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
                        <option :value="1">Slot 1</option>
                        <option :value="2">Slot 2</option>
                    </select>
                </div>
                 <div v-if="modalChannel.Type.includes('Digital')">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Rx Group List</label>
                    <input v-model="modalChannel.RxGroup" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none" />
                </div>
                 <div v-if="modalChannel.Type.includes('Digital')" class="col-span-2">
                    <label class="block text-xs font-medium text-slate-400 mb-1">Tx Contact (Talkgroup)</label>
                    <select v-model="modalChannel.ContactID" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 focus:outline-none">
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
                     <button @click="navigateChannel(-1)" class="px-3 py-2 rounded-lg bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white" title="Previous Channel">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>
                    </button>
                    <button @click="navigateChannel(1)" class="px-3 py-2 rounded-lg bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white" title="Next Channel">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
                    </button>
                </div>
                <div class="flex gap-3">
                    <button @click="showModal = false" class="px-4 py-2 rounded-lg bg-slate-800 text-slate-300 hover:bg-slate-700">Cancel</button>
                    <button @click="saveModal" class="px-4 py-2 rounded-lg bg-indigo-600 text-white hover:bg-indigo-500">Save</button>
                </div>
            </div>
        </div>
    </div>
  </div>
</template>
