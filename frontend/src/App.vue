<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import ZoneEditor from './components/ZoneEditor.vue'

interface Channel {
  ID: number
  Name: string
  RxFrequency: number
  TxFrequency: number
  Mode: string
  Tone: string
  Skip: boolean
  SquelchType: string
  RxTone: string
  TxTone: string
  RxDCS: string
  TxDCS: string
  Type: string
  Protocol: string
  ColorCode: number
  TimeSlot: number
  ContactID?: number
}

interface Contact {
  ID: number
  Name: string
  Callsign: string
  City: string
  State: string
  Country: string
  Remarks: string
  DMRID: number
  Type: string
  Source: string
}

interface Zone {
  ID: number
  Name: string
  Channels: Channel[]
}

const channels = ref<Channel[]>([])
const zones = ref<Zone[]>([])
const activeTab = ref('channels') // 'channels', 'talkgroups', 'digital', 'zones'
const searchQuery = ref('')
const showModal = ref(false)
const showContactModal = ref(false)
const editingChannel = ref<Channel | null>(null)
const editingContact = ref<Contact | null>(null)
const editingZone = ref<Zone | null>(null)
const showZoneModal = ref(false)
const viewingDigitalContact = ref<Contact | null>(null)
const showDigitalModal = ref(false)
const showRawData = ref(false)

// Contacts State (Split)
const talkgroups = ref<Contact[]>([]) // User contacts (loaded all at once for now, or paginated if needed, usually small)
const digitalContacts = ref<Contact[]>([]) // RadioID contacts (paginated)
const digitalTotal = ref(0)
const digitalPage = ref(1)
const digitalSearch = ref('')
const digitalLoading = ref(false)

const fetchChannels = async () => {
  try {
    const res = await fetch('/api/channels')
    channels.value = await res.json()
  } catch (e) {
    console.error("Failed to fetch channels", e)
  }
}


const fetchZones = async () => {
  try {
    const res = await fetch('/api/zones')
    zones.value = await res.json()
  } catch (e) {
    console.error("Failed to fetch zones", e)
  }
}

const filteredChannels = computed(() => {
  return channels.value.filter(ch => 
    ch.Name.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

// Fetch USER contacts (Talkgroups) - No pagination for now (assuming < 1000)
const fetchTalkgroups = async () => {
  try {
    const res = await fetch('/api/contacts?source=User&limit=1000')
    const data = await res.json()
    talkgroups.value = data.data || [] // Handle new API structure
  } catch (e) {
    console.error("Failed to fetch talkgroups", e)
  }
}

// Fetch DIGITAL contacts (RadioID) - Paginated
const digitalSort = ref('name')
const digitalOrder = ref('asc')

const fetchDigitalContacts = async () => {
  digitalLoading.value = true
  try {
    const params = new URLSearchParams({
       source: 'RadioID',
       page: digitalPage.value.toString(),
       limit: '50',
       search: digitalSearch.value,
       sort: digitalSort.value,
       order: digitalOrder.value
    })
    const res = await fetch(`/api/contacts?${params.toString()}`)
    const data = await res.json()
    digitalContacts.value = data.data || []
    digitalTotal.value = data.meta.total
  } catch (e) {
    console.error("Failed to fetch digital contacts", e)
  } finally {
    digitalLoading.value = false
  }
}

const sortDigital = (field: string) => {
  if (digitalSort.value === field) {
    digitalOrder.value = digitalOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    digitalSort.value = field
    digitalOrder.value = 'asc'
  }
  fetchDigitalContacts()
}

// Watchers for Digital Contacts Pagination
watch(digitalPage, () => fetchDigitalContacts())
// Debounce search
let searchTimeout: any
watch(digitalSearch, () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    digitalPage.value = 1
    fetchDigitalContacts()
  }, 300)
})
const openDigitalModal = (c: Contact) => {
  viewingDigitalContact.value = c
  showDigitalModal.value = true
  showRawData.value = false // Reset state
}

const closeDigitalModal = () => {
    showDigitalModal.value = false
    viewingDigitalContact.value = null
}


const openEditModal = (ch: Channel) => {
  editingChannel.value = { ...ch } // Clone
  showModal.value = true
}

const openAddModal = () => {
  editingChannel.value = {
    ID: 0,
    Name: '',
    RxFrequency: 146.5200,
    TxFrequency: 146.5200,
    Mode: 'FM',
    Tone: '',
    Skip: false,
    SquelchType: '',
    RxTone: '',
    TxTone: '',
    RxDCS: '',
    TxDCS: '',
    Type: 'Analog',
    Protocol: 'FM',
    ColorCode: 1,
    TimeSlot: 1,
    ContactID: undefined
  }
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  editingChannel.value = null
}

const openContactModal = (c: Contact | null) => {
  if (c) {
    editingContact.value = { ...c }
  } else {
    editingContact.value = {
      ID: 0,
      Name: '',
      Callsign: '',
      City: '',
      State: '',
      Country: '',
      Remarks: '',
      DMRID: 0,
      Type: 'Group',
      Source: 'User'
    }
  }
  showContactModal.value = true
}

const closeContactModal = () => {
  showContactModal.value = false
  editingContact.value = null
}

const saveContact = async () => {
  if (!editingContact.value) return
  if (!editingContact.value.Name || editingContact.value.DMRID <= 0) {
    alert("Name and valid DMR ID are required")
    return
  }
  
  try {
    const res = await fetch('/api/contacts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(editingContact.value)
    })
    if (res.ok) {
        // Refresh both lists to be safe
       fetchTalkgroups() 
       closeContactModal()
    } else {
      console.error("Failed to save contact")
    }
  } catch (e) {
    console.error("Error saving contact", e)
  }
}

const deleteChannel = async (id: number) => {
  if (!confirm("Are you sure you want to delete this channel?")) return
  try {
    await fetch(`/api/channels?id=${id}`, { method: 'DELETE' })
    await fetchChannels()
    if (editingChannel.value?.ID === id) closeModal()
  } catch (e) {
    console.error("Failed to delete channel", e)
  }
}

const deleteContact = async (id: number) => {
  if (!confirm("Are you sure you want to delete this contact?")) return
  try {
    const res = await fetch(`/api/contacts?id=${id}`, { method: 'DELETE' })
    if (!res.ok) {
        alert(await res.text())
        return
    }
    fetchTalkgroups()
  } catch (e) {
    console.error("Failed to delete contact", e)
  }
  }


const deleteZone = async (id: number) => {
  if (!confirm("Are you sure you want to delete this zone?")) return
  try {
    await fetch(`/api/zones?id=${id}`, { method: 'DELETE' })
    fetchZones()
  } catch (e) {
    console.error("Failed to delete zone", e)
  }
}

const openZoneModal = (z: Zone | null) => {
  if (z) {
    // Clone deeply to avoid mutating list directly before save
    editingZone.value = JSON.parse(JSON.stringify(z))
  } else {
    editingZone.value = { ID: 0, Name: '', Channels: [] }
  }
  showZoneModal.value = true
}

const closeZoneModal = () => {
  showZoneModal.value = false
  editingZone.value = null
}

const saveZone = async () => {
  if (!editingZone.value) return
  if (!editingZone.value.Name) {
    alert("Zone Name is required")
    return
  }

  try {
    // 1. Save Zone (Create/Update)
    const res = await fetch('/api/zones', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ID: editingZone.value.ID, Name: editingZone.value.Name }) // Send only basic info first
    })
    
    if (res.ok) {
      const savedZone = await res.json()
      
      // 2. Assign Channels (Order)
      // Extract IDs in order
      const channelIDs = editingZone.value.Channels.map(c => c.ID)
      const assignRes = await fetch(`/api/zones/assign?id=${savedZone.ID}`, {
         method: 'POST',
         headers: { 'Content-Type': 'application/json' },
         body: JSON.stringify(channelIDs)
      })

      if (assignRes.ok) {
         fetchZones()
         closeZoneModal()
      } else {
         console.error("Failed to assign channels")
      }
    } else {
      console.error("Failed to save zone")
    }
  } catch (e) {
    console.error("Error saving zone", e)
  }
}

const updateRxFrequency = (e: Event) => {
  if (editingChannel.value) {
    editingChannel.value.RxFrequency = parseFloat(parseFloat((e.target as HTMLInputElement).value).toFixed(4))
  }
}

const updateTxFrequency = (e: Event) => {
  if (editingChannel.value) {
    editingChannel.value.TxFrequency = parseFloat(parseFloat((e.target as HTMLInputElement).value).toFixed(4))
  }
}

const tones = [
  "67.0", "69.3", "71.9", "74.4", "77.0", "79.7", "82.5", "85.4", "88.5", "91.5",
  "94.8", "97.4", "100.0", "103.5", "107.2", "110.9", "114.8", "118.8", "123.0", "127.3",
  "131.8", "136.5", "141.3", "146.2", "151.4", "156.7", "162.2", "167.9", "173.8", "179.9",
  "186.2", "192.8", "203.5", "210.7", "218.1", "225.7", "233.6", "241.8", "250.3", "254.1"
]

const dcsCodes = [
  "023", "025", "026", "031", "032", "036", "043", "047", "051", "053", "054", "065",
  "071", "072", "073", "074", "114", "115", "116", "122", "125", "131", "132", "134",
  "143", "145", "152", "155", "156", "162", "165", "172", "174", "205", "212", "223",
  "225", "226", "243", "244", "245", "246", "251", "252", "255", "261", "263", "265",
  "266", "271", "274", "306", "311", "315", "325", "331", "332", "343", "346", "351",
  "356", "364", "365", "371", "411", "412", "413", "423", "431", "432", "445", "446",
  "452", "454", "455", "462", "464", "465", "466", "503", "506", "516", "523", "526",
  "532", "546", "565", "606", "612", "624", "627", "631", "632", "654", "662", "664",
  "703", "712", "723", "731", "732", "734", "743", "754"
]

const saveChannel = async () => {
  if (!editingChannel.value) return

  try {
    const res = await fetch('/api/channels', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(editingChannel.value)
    })
    if (res.ok) {
      await fetchChannels()
      closeModal()
    } else {
      console.error("Failed to save channel")
    }
  } catch (e) {
    console.error("Error saving channel", e)
  }
}



// Watch Protocol changes to update Mode automatically
watch(() => editingChannel.value?.Protocol, (newProtocol) => {
  if (editingChannel.value && newProtocol) {
    editingChannel.value.Mode = newProtocol
  }
})

// Computed properties for formatted frequency display/editing
const editingRxFrequency = computed({
  get: () => editingChannel.value?.RxFrequency.toFixed(4) || "0.0000",
  set: (val: string) => {
    if (editingChannel.value) {
      editingChannel.value.RxFrequency = parseFloat(parseFloat(val).toFixed(4))
    }
  }
})

const editingTxFrequency = computed({
  get: () => editingChannel.value?.TxFrequency.toFixed(4) || "0.0000",
  set: (val: string) => {
    if (editingChannel.value) {
      editingChannel.value.TxFrequency = parseFloat(parseFloat(val).toFixed(4))
    }
  }
})

onMounted(() => {
  fetchChannels()
  fetchChannels()
  fetchZones()
  fetchTalkgroups()
  fetchDigitalContacts()
  connectWebSocket()
})

const fileInput = ref<HTMLInputElement | null>(null)

// Import Modal State
const showImportModal = ref(false)
const importSource = ref('channels') // 'channels' or 'radioid'
const importOverwrite = ref(false)
const importUseFilter = ref(false)
const importAutoDownload = ref(false)
const selectedImportFile = ref<File | null>(null)
const selectedFilterFile = ref<File | null>(null)
const isImporting = ref(false)

// WebSocket Progress
const importProgress = ref({
  total: 0,
  processed: 0,
  status: 'idle',
  message: ''
})

const connectWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/ws`)

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data)
            if (msg.type === 'import_progress') {
                importProgress.value = msg.data
                // Automatically stop loading state if complete
                if (importProgress.value.status === 'completed' || importProgress.value.status === 'error') {
                    isImporting.value = false
                    if (importProgress.value.status === 'completed' && showImportModal.value) {
                         // Optional: auto-close or just show success
                         alert(importProgress.value.message)
                         showImportModal.value = false
                         fetchChannels() // Refresh after import
                         fetchDigitalContacts()
                    }
                } else if (importProgress.value.status === 'running') {
                    isImporting.value = true
                }
            }
        } catch (e) {
            console.error("WS Parse Error", e)
        }
    }

    ws.onclose = () => {
        // Reconnect logic?
        setTimeout(connectWebSocket, 3000)
    }
}

// Export Modal State
const showExportModal = ref(false)
const exportFormat = ref('csv') // 'csv' (uses radio param logic)
const exportRadio = ref('db25d') // 'db25d', 'dm32uv'
const exportZoneID = ref<number | null>(null)
const exportUseFirstName = ref(false)

const openImportModal = () => {
  importSource.value = 'channels'
  importOverwrite.value = false
  importUseFilter.value = false
  importAutoDownload.value = false
  selectedImportFile.value = null
  selectedFilterFile.value = null
  showImportModal.value = true
}

const handleImportFileChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedImportFile.value = target.files[0]
  }
}

const handleFilterFileChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedFilterFile.value = target.files[0]
  }
}

const submitImport = async () => {
  const formData = new FormData()

  if (importSource.value === 'radioid') {
      formData.append('format', 'radioid')
      formData.append('overwrite', importOverwrite.value ? 'true' : 'false')
      
      if (importAutoDownload.value) {
          formData.append('source_mode', 'download')
      } else {
         if (!selectedImportFile.value) {
            alert("Please select a file or choose Auto Download")
            return
         }
         formData.append('file', selectedImportFile.value)
      }

      if (importUseFilter.value && selectedFilterFile.value) {
        formData.append('filter_file', selectedFilterFile.value)
      }
  } else if (importSource.value === 'zip') {
      formData.append('format', 'zip')
       if (!selectedImportFile.value) {
        alert("Please select a zip file")
        return
      }
      formData.append('file', selectedImportFile.value)
  } else {
    if (!selectedImportFile.value) {
        alert("Please select a file to import")
        return
    }
    formData.append('file', selectedImportFile.value)
    formData.append('format', 'generic')
  }

  isImporting.value = true
  try {
    const res = await fetch('/api/import', {
      method: 'POST',
      body: formData
    })
    if (res.ok) {
      const result = await res.json()
      await fetchChannels()
      if (importSource.value === 'radioid') await fetchDigitalContacts()
      
      alert(`Import successful!\nImported: ${result.imported}\nSkipped: ${result.skipped}`)
      showImportModal.value = false
    } else {
      const errText = await res.text()
      alert("Import failed: " + errText)
    }
  } catch (e) {
    console.error("Import error", e)
    alert("Import error")
  } finally {
    isImporting.value = false
  }
}

// ... existing code ...

// Template update for button
/*
<div class="p-6 border-t border-slate-800 flex justify-end gap-3">
    <button @click="showImportModal = false" :disabled="isImporting" class="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium transition-colors disabled:opacity-50">Cancel</button>
    <button @click="submitImport" :disabled="isImporting" class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg shadow-indigo-500/20 transition-colors disabled:opacity-50 flex items-center gap-2">
        <span v-if="isImporting">
            <svg class="animate-spin h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
        </span>
        {{ isImporting ? 'Importing...' : 'Import' }}
    </button>
</div>
*/

const openExportModal = () => {
    exportFormat.value = 'csv'
    exportRadio.value = 'db25d'
    exportZoneID.value = null
    showExportModal.value = true
}

const submitExport = () => {
  let url = `/api/export?radio=${exportRadio.value}`
  if (exportRadio.value === 'dm32uv') {
      // defaults to zip
  } else {
      url += `&format=db25d` // explicit for now
  }
  
  if (exportZoneID.value) {
      url += `&zone_id=${exportZoneID.value}`
  }
  window.location.href = url
  showExportModal.value = false
}
</script>

<template>
  <div class="min-h-screen bg-slate-900 text-slate-100 font-sans selection:bg-indigo-500 selection:text-white">
    <!-- Background Gradient Mesh -->
    <div class="fixed inset-0 z-0 pointer-events-none">
      <div class="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] rounded-full bg-indigo-600/20 blur-[120px]"></div>
      <div class="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] rounded-full bg-fuchsia-600/20 blur-[120px]"></div>
    </div>

    <div class="relative z-10 container mx-auto p-6 max-w-7xl">
      <!-- Header -->
      <header class="flex justify-between items-center mb-10">
        <div>
          <h1 class="text-4xl font-extrabold tracking-tight bg-gradient-to-r from-indigo-400 to-fuchsia-400 bg-clip-text text-transparent">
            Codeplug Editor
          </h1>
          <p class="text-slate-400 mt-1">Manage your radio channels with style.</p>
        </div>
        <div class="flex gap-3">
           <button @click="openImportModal" class="px-4 py-2 rounded-lg bg-slate-800/50 hover:bg-slate-700/50 border border-slate-700/50 backdrop-blur-md transition-all text-sm font-medium">
            Import CSV
          </button>
          <button @click="openExportModal" class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 shadow-lg shadow-indigo-500/20 transition-all text-sm font-medium">
            Export Codeplug
          </button>
        </div>
      </header>

      <!-- Stats / Quick Actions -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-10">
        <div class="p-6 rounded-2xl bg-slate-800/40 border border-slate-700/50 backdrop-blur-xl shadow-xl">
          <div class="text-slate-400 text-sm font-medium mb-1">Total Channels</div>
          <div class="text-3xl font-bold text-white">{{ channels.length }}</div>
        </div>
        <div class="p-6 rounded-2xl bg-slate-800/40 border border-slate-700/50 backdrop-blur-xl shadow-xl">
          <div class="text-slate-400 text-sm font-medium mb-1">Total Zones</div>
          <div class="text-3xl font-bold text-white">{{ zones.length }}</div>
        </div>
        <div @click="activeTab === 'channels' ? openAddModal() : (activeTab === 'zones' ? openZoneModal(null) : openContactModal(null))" class="p-6 rounded-2xl bg-slate-800/40 border border-slate-700/50 backdrop-blur-xl shadow-xl flex items-center justify-center border-dashed border-2 border-slate-700 hover:border-indigo-500/50 hover:bg-slate-800/60 transition-all cursor-pointer group">
          <div class="text-center">
            <div class="text-indigo-400 group-hover:scale-110 transition-transform mb-2">
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="mx-auto"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
            </div>

            <span class="text-sm font-medium text-slate-300">{{ activeTab === 'channels' ? 'Add Channel' : (activeTab === 'zones' ? 'Add Zone' : 'Add Contact') }}</span>
          </div>

        </div>
      </div>

      <!-- Tabs -->
      <div class="flex gap-4 mb-6">
        <button 
          @click="activeTab = 'channels'" 
          class="px-4 py-2 rounded-lg font-medium transition-colors"
          :class="activeTab === 'channels' ? 'bg-indigo-600 text-white' : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'"
        >
          Channels
        </button>
         <button 
          @click="activeTab = 'talkgroups'" 
          class="px-4 py-2 rounded-lg font-medium transition-colors"
          :class="activeTab === 'talkgroups' ? 'bg-indigo-600 text-white' : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'"
        >
          Contacts
        </button>
        <button 
          @click="activeTab = 'zones'" 
          class="px-4 py-2 rounded-lg font-medium transition-colors"
          :class="activeTab === 'zones' ? 'bg-indigo-600 text-white' : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'"
        >
          Zones
        </button>
         <button  
          @click="activeTab = 'digital'" 
          class="px-4 py-2 rounded-lg font-medium transition-colors"
          :class="activeTab === 'digital' ? 'bg-indigo-600 text-white' : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'"
        >
          Digital Contact List
        </button>
      </div>

      <!-- Main Content -->
      <div class="rounded-3xl bg-slate-900/60 border border-slate-800 backdrop-blur-xl shadow-2xl overflow-hidden min-h-[400px]">
        <!-- Toolbar -->
        <div class="p-4 border-b border-slate-800 flex items-center gap-4">
          <div class="relative flex-1 max-w-md">
            <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-slate-500">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
            </div>
            
            <input 
              v-if="activeTab === 'channels'"
              v-model="searchQuery"
              type="text" 
              placeholder="Search channels..." 
              class="w-full pl-10 pr-4 py-2 bg-slate-950/50 border border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500/50 text-sm placeholder-slate-500 transition-all"
            >
             <input 
              v-if="activeTab === 'digital'"
              v-model="digitalSearch"
              type="text" 
              placeholder="Search by Name, Callsign, or ID..." 
              class="w-full pl-10 pr-4 py-2 bg-slate-950/50 border border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500/50 text-sm placeholder-slate-500 transition-all"
            >
            <div v-if="activeTab === 'talkgroups'" class="text-slate-400 text-sm italic pl-2">
                Contacts used for Channel Programming (TX) and RX Groups
            </div>
          </div>
          <div class="flex gap-2" v-if="activeTab === 'digital'">
             <span v-if="digitalLoading" class="text-sm text-slate-400 flex items-center">Loading...</span>
             <span class="text-sm text-slate-400 flex items-center">
                 Page {{ digitalPage }} of {{ Math.ceil(digitalTotal / 50) }}
             </span>
             <button @click="digitalPage--" :disabled="digitalPage <= 1" class="px-3 py-1 rounded bg-slate-800 text-slate-300 disabled:opacity-50">Prev</button>
             <button @click="digitalPage++" :disabled="digitalPage >= Math.ceil(digitalTotal / 50)" class="px-3 py-1 rounded bg-slate-800 text-slate-300 disabled:opacity-50">Next</button>
          </div>
        </div>

        <!-- Channels Table -->
        <div v-if="activeTab === 'channels'" class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
            <thead>
              <tr class="border-b border-slate-800 text-xs uppercase tracking-wider text-slate-500 font-semibold">
                <th class="px-6 py-4">ID</th>
                <th class="px-6 py-4">Name</th>
                <th class="px-6 py-4">Frequency</th>
                <th class="px-6 py-4">Mode</th>
                <th class="px-6 py-4">Tone</th>
                <th class="px-6 py-4">Skip</th>
                <th class="px-6 py-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/50">
              <tr v-for="ch in filteredChannels" :key="ch.ID" @click="openEditModal(ch)" class="group hover:bg-slate-800/30 transition-colors cursor-pointer">
                <td class="px-6 py-4 text-slate-500 font-mono text-xs">#{{ ch.ID }}</td>
                <td class="px-6 py-4 font-medium text-slate-200">{{ ch.Name }}</td>
                <td class="px-6 py-4">
                  <div class="flex flex-col">
                    <span class="text-indigo-300 font-mono">{{ ch.RxFrequency.toFixed(4) }}</span>
                    <span class="text-xs text-slate-500 font-mono" v-if="ch.RxFrequency !== ch.TxFrequency">TX: {{ ch.TxFrequency.toFixed(4) }}</span>
                  </div>
                </td>
                <td class="px-6 py-4">
                  <span class="px-2 py-1 rounded-md text-xs font-medium border"
                    :class="{
                      'bg-emerald-500/10 text-emerald-400 border-emerald-500/20': ch.Mode === 'FM' || ch.Mode === 'NFM',
                      'bg-blue-500/10 text-blue-400 border-blue-500/20': ch.Mode === 'DMR',
                      'bg-purple-500/10 text-purple-400 border-purple-500/20': ch.Mode === 'DN' || ch.Mode === 'DV'
                    }"
                  >
                    {{ ch.Mode }}
                  </span>
                </td>
                <td class="px-6 py-4 text-slate-400 text-sm">{{ ch.Tone || '-' }}</td>
                <td class="px-6 py-4">
                  <span v-if="ch.Skip" class="px-2 py-1 rounded-full bg-red-500/10 text-red-400 text-xs border border-red-500/20">Skipped</span>
                  <span v-else class="text-slate-600 text-xs">-</span>
                </td>
                <td class="px-6 py-4 text-right">
                  <button @click.stop="openEditModal(ch)" class="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                  </button>
                  <button @click.stop="deleteChannel(ch.ID)" class="p-2 hover:bg-red-900/30 rounded-lg text-slate-400 hover:text-red-400 transition-colors ml-1">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                  </button>
                </td>
              </tr>
              <tr v-if="filteredChannels.length === 0">
                <td colspan="8" class="px-6 py-12 text-center text-slate-500">
                  No channels found matching your search.
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Talkgroups Table (User Contacts) -->
        <div v-if="activeTab === 'talkgroups'" class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
            <thead>
              <tr class="border-b border-slate-800 text-xs uppercase tracking-wider text-slate-500 font-semibold">
                <th class="px-6 py-4">Name</th>
                <th class="px-6 py-4">Type</th>
                <th class="px-6 py-4">DMR ID</th>
                <th class="px-6 py-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/50">
              <tr v-for="c in talkgroups" :key="c.ID" @click="openContactModal(c)" class="group hover:bg-slate-800/30 transition-colors cursor-pointer">
                <td class="px-6 py-4 font-medium text-slate-200">{{ c.Name }}</td>
                <td class="px-6 py-4">
                  <span class="px-2 py-1 rounded-md text-xs font-medium border"
                    :class="{
                      'bg-indigo-500/10 text-indigo-400 border-indigo-500/20': c.Type === 'Group',
                      'bg-orange-500/10 text-orange-400 border-orange-500/20': c.Type === 'Private',
                      'bg-slate-500/10 text-slate-400 border-slate-500/20': c.Type === 'AllCall'
                    }"
                  >
                    {{ c.Type }}
                  </span>
                </td>
                <td class="px-6 py-4 text-slate-400 font-mono">{{ c.DMRID }}</td>
                <td class="px-6 py-4 text-right">
                   <button @click.stop="openContactModal(c)" class="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                  </button>
                  <button @click.stop="deleteContact(c.ID)" class="p-2 hover:bg-red-900/30 rounded-lg text-slate-400 hover:text-red-400 transition-colors ml-1">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                  </button>
                </td>
              </tr>
              <tr v-if="talkgroups.length === 0">
                 <td colspan="4" class="px-6 py-12 text-center text-slate-500">
                  No contacts found. Add one!
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Digital Contacts Table (RadioID) -->
        <div v-if="activeTab === 'digital'" class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
            <thead>
              <tr class="border-b border-slate-800 text-xs uppercase tracking-wider text-slate-500 font-semibold">
                <th class="px-6 py-4 cursor-pointer hover:text-white select-none" @click="sortDigital('dmr_id')">
                    ID
                    <span v-if="digitalSort === 'dmr_id'" class="ml-1 text-indigo-400">{{ digitalOrder === 'asc' ? '↑' : '↓' }}</span>
                </th>
                <th class="px-6 py-4 cursor-pointer hover:text-white select-none" @click="sortDigital('callsign')">
                    Callsign
                     <span v-if="digitalSort === 'callsign'" class="ml-1 text-indigo-400">{{ digitalOrder === 'asc' ? '↑' : '↓' }}</span>
                </th>
                <th class="px-6 py-4 cursor-pointer hover:text-white select-none" @click="sortDigital('name')">
                    Name
                     <span v-if="digitalSort === 'name'" class="ml-1 text-indigo-400">{{ digitalOrder === 'asc' ? '↑' : '↓' }}</span>
                </th>
                <th class="px-6 py-4">Location</th>
                <th class="px-6 py-4">Type</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/50">
              <tr v-for="c in digitalContacts" :key="c.ID" @click="openDigitalModal(c)" class="group hover:bg-slate-800/30 transition-colors cursor-pointer">
                <td class="px-6 py-4 text-slate-400 font-mono">{{ c.DMRID }}</td>
                <td class="px-6 py-4 font-mono text-indigo-300 font-bold">{{ c.Callsign }}</td>
                <td class="px-6 py-4 font-medium text-slate-200">{{ c.Name }}</td>
                <td class="px-6 py-4 text-slate-400 text-sm">
                    {{ c.City }}<span v-if="c.State">, {{ c.State }}</span><span v-if="c.Country">, {{ c.Country }}</span>
                </td>
                <td class="px-6 py-4">
                  <span class="px-2 py-1 rounded-md text-xs font-medium border"
                    :class="{
                      'bg-indigo-500/10 text-indigo-400 border-indigo-500/20': c.Type === 'Group',
                      'bg-orange-500/10 text-orange-400 border-orange-500/20': c.Type === 'Private',
                      'bg-slate-500/10 text-slate-400 border-slate-500/20': c.Type === 'AllCall'
                    }"
                  >
                    {{ c.Type }}
                  </span>
                </td>
              </tr>
              <tr v-if="digitalContacts.length === 0">
                 <td colspan="3" class="px-6 py-12 text-center text-slate-500">
                  <span v-if="digitalLoading">Loading...</span>
                  <span v-else>No digital contact list found. Import from RadioID.net!</span>
                </td>
              </tr>
            </tbody>
          </table>

        </div>

        <!-- Zones Table -->
        <div v-if="activeTab === 'zones'" class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
            <thead>
              <tr class="border-b border-slate-800 text-xs uppercase tracking-wider text-slate-500 font-semibold">
                <th class="px-6 py-4">Name</th>
                <th class="px-6 py-4">Channels</th>
                <th class="px-6 py-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/50">
              <tr v-for="z in zones" :key="z.ID" @click="openZoneModal(z)" class="group hover:bg-slate-800/30 transition-colors cursor-pointer">
                <td class="px-6 py-4 font-medium text-slate-200">{{ z.Name }}</td>
                <td class="px-6 py-4 text-slate-400 text-sm">
                   {{ z.Channels ? z.Channels.length : 0 }} channels
                </td>
                <td class="px-6 py-4 text-right">
                   <button @click.stop="openZoneModal(z)" class="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                  </button>
                  <button @click.stop="deleteZone(z.ID)" class="p-2 hover:bg-red-900/30 rounded-lg text-slate-400 hover:text-red-400 transition-colors ml-1">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                  </button>
                </td>
              </tr>
              <tr v-if="zones.length === 0">
                 <td colspan="3" class="px-6 py-12 text-center text-slate-500">
                  No zones found. Create one!
                </td>
              </tr>
            </tbody>
          </table>
        </div>

      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="showModal && editingChannel" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div class="bg-slate-900 border border-slate-700 rounded-2xl shadow-2xl w-full max-w-lg overflow-hidden">
        <div class="p-6 border-b border-slate-800 flex justify-between items-center">
          <h2 class="text-xl font-bold text-white">{{ editingChannel.ID === 0 ? 'Add Channel' : 'Edit Channel' }}</h2>
          <button @click="closeModal" class="text-slate-400 hover:text-white">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
          </button>
        </div>
        <div class="p-6 space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-400 mb-1">Name</label>
            <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">RX Frequency</label>
              <input 
                v-model.lazy="editingRxFrequency"
                type="text" 
                class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white font-mono"
              >
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">TX Frequency</label>
               <input 
                v-model.lazy="editingTxFrequency"
                type="text" 
                class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white font-mono"
              >
            </div>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">Channel Type</label>
              <select v-model="editingChannel.Type" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
                <option value="Analog">Analog</option>
                <option value="Digital">Digital</option>
                <option value="Mixed">Mixed</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">Protocol</label>
              <select v-model="editingChannel.Protocol" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
                <option v-if="editingChannel.Type === 'Analog' || editingChannel.Type === 'Mixed'" value="FM">FM</option>
                <option v-if="editingChannel.Type === 'Digital' || editingChannel.Type === 'Mixed'" value="DMR">DMR</option>
                <option v-if="editingChannel.Type === 'Digital' || editingChannel.Type === 'Mixed'" value="Fusion">Fusion</option>
                <option v-if="editingChannel.Type === 'Digital' || editingChannel.Type === 'Mixed'" value="D-Star">D-Star</option>
                <option v-if="editingChannel.Type === 'Digital' || editingChannel.Type === 'Mixed'" value="NXDN">NXDN</option>
              </select>
            </div>
          </div>

          <div v-if="editingChannel.Protocol === 'DMR'" class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">Contact (Talkgroup)</label>
              <select v-model.number="editingChannel.ContactID" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
                <option :value="undefined">None</option>
                <option v-for="c in talkgroups" :key="c.ID" :value="c.ID">{{ c.Name }}</option>
              </select>
            </div>
          </div>

          <div v-if="editingChannel.Protocol === 'DMR'" class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">Color Code</label>
              <input v-model.number="editingChannel.ColorCode" type="number" min="0" max="15" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">Time Slot</label>
               <select v-model.number="editingChannel.TimeSlot" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
                 <option :value="1">Slot 1</option>
                 <option :value="2">Slot 2</option>
              </select>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">Squelch Type</label>
              <select v-model="editingChannel.SquelchType" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
                <option value="">None</option>
                <option value="Tone">Tone (TX Only)</option>
                <option value="TSQL">TSQL (TX & RX)</option>
                <option value="DCS">DCS</option>
              </select>
            </div>
          </div>
          
          <div v-if="editingChannel.SquelchType === 'Tone' || editingChannel.SquelchType === 'TSQL'" class="grid grid-cols-2 gap-4">
             <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">TX Tone</label>
              <select v-model="editingChannel.TxTone" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
                 <option v-for="tone in tones" :key="tone" :value="tone">{{ tone }}</option>
              </select>
            </div>
            <div v-if="editingChannel.SquelchType === 'TSQL'">
              <label class="block text-sm font-medium text-slate-400 mb-1">RX Tone</label>
              <select v-model="editingChannel.RxTone" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
                 <option v-for="tone in tones" :key="tone" :value="tone">{{ tone }}</option>
              </select>
            </div>
          </div>

          <div v-if="editingChannel.SquelchType === 'DCS'" class="grid grid-cols-2 gap-4">
             <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">TX DCS</label>
              <select v-model="editingChannel.TxDCS" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
                 <option v-for="code in dcsCodes" :key="code" :value="code">{{ code }}</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-400 mb-1">RX DCS</label>
              <select v-model="editingChannel.RxDCS" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
                 <option v-for="code in dcsCodes" :key="code" :value="code">{{ code }}</option>
              </select>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <input v-model="editingChannel.Skip" type="checkbox" id="skip" class="w-4 h-4 rounded border-slate-700 bg-slate-950 text-indigo-600 focus:ring-indigo-500">
            <label for="skip" class="text-sm font-medium text-slate-300">Skip Export</label>
          </div>
        </div>
        <div class="p-6 border-t border-slate-800 flex justify-end gap-3">
          <button @click="closeModal" class="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium transition-colors">Cancel</button>
          <button @click="saveChannel" class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg shadow-indigo-500/20 transition-colors">Save Changes</button>
        </div>
      </div>
    </div>

    <!-- Contact Modal (User Talkgroups) -->
    <div v-if="showContactModal && editingContact" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div class="bg-slate-900 border border-slate-700 rounded-2xl shadow-2xl w-full max-w-sm overflow-hidden">
        <div class="p-6 border-b border-slate-800 flex justify-between items-center">
          <h2 class="text-xl font-bold text-white">{{ editingContact.ID === 0 ? 'Add Contact' : 'Edit Contact' }}</h2>
          <button @click="closeContactModal" class="text-slate-400 hover:text-white">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
          </button>
        </div>
        <div class="p-6 space-y-4">
           <div>
            <label class="block text-sm font-medium text-slate-400 mb-1">Name</label>
            <input v-model="editingContact.Name" type="text" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-400 mb-1">DMR ID</label>
            <input v-model.number="editingContact.DMRID" type="number" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
          </div>
           <div>
            <label class="block text-sm font-medium text-slate-400 mb-1">Type</label>
            <select v-model="editingContact.Type" class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white">
              <option value="Group">Group Call</option>
              <option value="Private">Private Call</option>
              <option value="AllCall">All Call</option>
            </select>
          </div>
        </div>
         <div class="p-6 border-t border-slate-800 flex justify-end gap-3">
          <button @click="closeContactModal" class="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium transition-colors">Cancel</button>
          <button @click="saveContact" class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg shadow-indigo-500/20 transition-colors">Save Talkgroup</button>
        </div>
      </div>
    </div>

    <!-- Import Modal -->
    <div v-if="showImportModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div class="bg-slate-900 border border-slate-700 rounded-2xl shadow-2xl w-full max-w-lg overflow-hidden">
        <div class="p-6 border-b border-slate-800 flex justify-between items-center">
            <h2 class="text-xl font-bold text-white">Import Data</h2>
            <button @click="showImportModal = false" class="text-slate-400 hover:text-white">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
            </button>
        </div>
        <div class="p-6 space-y-4">
             <div class="mb-4">
               <label class="block text-sm font-medium text-slate-400 mb-2">Import Source</label>
               <div class="flex gap-4">
                 <label class="flex items-center gap-2 cursor-pointer">
                   <input type="radio" v-model="importSource" value="channels" class="text-indigo-600 focus:ring-indigo-500">
                   <span class="text-white">Generic/DB25-D CSV</span>
                 </label>
                  <label class="flex items-center gap-2 cursor-pointer">
                   <input type="radio" v-model="importSource" value="zip" class="text-indigo-600 focus:ring-indigo-500">
                   <span class="text-white">Full Backup (Zip)</span>
                 </label>
                 <label class="flex items-center gap-2 cursor-pointer">
                   <input type="radio" v-model="importSource" value="radioid" class="text-indigo-600 focus:ring-indigo-500">
                   <span class="text-white">RadioID.net</span>
                 </label>
               </div>
            </div>

            <!-- ZIP Input -->
            <div v-if="importSource === 'zip'" class="mb-4">
              <label class="block text-sm font-medium text-slate-400 mb-1">Select Zip File</label>
              <input type="file" ref="fileInput" @change="handleImportFileChange" accept=".zip" class="block w-full text-sm text-slate-400 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-semibold file:bg-slate-800 file:text-indigo-400 hover:file:bg-slate-700"/>
            </div>

            <div v-if="importSource === 'channels'" class="border-t border-slate-800 pt-4">
                <label class="block text-sm font-medium text-slate-400 mb-1">File to Import</label>
                <input type="file" @change="handleImportFileChange" accept=".csv" class="block w-full text-sm text-slate-400 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-semibold file:bg-slate-800 file:text-indigo-400 hover:file:bg-slate-700"/>
            </div>

            <div v-if="importSource === 'radioid'" class="space-y-4 border-t border-slate-800 pt-4 animate-in fade-in slide-in-from-top-2">
                
                <div class="flex items-center gap-2">
                    <input v-model="importAutoDownload" type="checkbox" id="autoDownload" class="w-4 h-4 rounded border-slate-700 bg-slate-950 text-indigo-600 focus:ring-indigo-500">
                    <label for="autoDownload" class="text-sm font-medium text-slate-300">Auto-Download from RadioID.net</label>
                </div>
                <p v-if="importAutoDownload" class="text-xs text-indigo-400 ml-6">Will download latest user.csv (~100MB+)</p>

                <div class="flex items-center gap-2">
                    <input v-model="importOverwrite" type="checkbox" id="overwrite" class="w-4 h-4 rounded border-slate-700 bg-slate-950 text-indigo-600 focus:ring-indigo-500">
                    <label for="overwrite" class="text-sm font-medium text-slate-300">Overwrite Existing Digital Contacts</label>
                </div>

                <div class="flex items-center gap-2">
                    <input v-model="importUseFilter" type="checkbox" id="useFilter" class="w-4 h-4 rounded border-slate-700 bg-slate-950 text-indigo-600 focus:ring-indigo-500">
                    <label for="useFilter" class="text-sm font-medium text-slate-300">Filter by Brandmeister Last Heard?</label>
                </div>

                <div v-if="importUseFilter" class="pl-6 pt-2">
                     <label class="block text-sm font-medium text-slate-400 mb-1">Last Heard CSV</label>
                     <input type="file" @change="handleFilterFileChange" accept=".csv" class="block w-full text-sm text-slate-400 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-semibold file:bg-slate-800 file:text-indigo-400 hover:file:bg-slate-700"/>
                </div>
            </div>

            <!-- Progress Bar -->
            <div v-if="isImporting && importProgress.status === 'running'" class="border-t border-slate-800 pt-4 animate-in fade-in">
                <div class="flex justify-between text-xs text-slate-400 mb-1">
                    <span>{{ importProgress.message }}</span>
                    <span>{{ Math.round((importProgress.processed / (importProgress.total || 1)) * 100) }}%</span>
                </div>
                <div class="h-2 bg-slate-800 rounded-full overflow-hidden">
                    <div class="h-full bg-indigo-500 transition-all duration-300 ease-out" :style="{ width: `${(importProgress.processed / (importProgress.total || 1)) * 100}%` }"></div>
                </div>
                <div class="text-xs text-slate-500 mt-1 text-right">{{ importProgress.processed }} / {{ importProgress.total }}</div>
            </div>
        </div>
         <div class="p-6 border-t border-slate-800 flex justify-end gap-3">
            <button @click="showImportModal = false" :disabled="isImporting" class="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium transition-colors disabled:opacity-50">Cancel</button>
            <button @click="submitImport" :disabled="isImporting" class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg shadow-indigo-500/20 transition-colors disabled:opacity-50 flex items-center gap-2">
                <span v-if="isImporting">
                    <svg class="animate-spin h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                </span>
                {{ isImporting ? 'Importing...' : 'Import' }}
            </button>
        </div>
      </div>
    </div>


    <!-- Export Modal -->
    <div v-if="showExportModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div class="bg-slate-900 border border-slate-700 rounded-2xl shadow-2xl w-full max-w-sm overflow-hidden">
         <div class="p-6 border-b border-slate-800 flex justify-between items-center">
            <h2 class="text-xl font-bold text-white">Export Codeplug</h2>
            <button @click="showExportModal = false" class="text-slate-400 hover:text-white">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
            </button>
        </div>
        <div class="p-6 space-y-4">

            <div class="mt-4 pt-4 border-t border-slate-800">
               <label class="flex items-center gap-2">
                  <input v-model="exportUseFirstName" type="checkbox" class="w-4 h-4 rounded border-slate-700 bg-slate-950 text-indigo-600 focus:ring-indigo-500">
                  <span class="text-sm text-slate-300">Use First Name for Contacts</span>
               </label>
               <p class="text-xs text-slate-500 ml-6 mt-1">If unchecked, full names will be used.</p>
            </div>
          </div>

         <div class="p-6 border-t border-slate-800 flex justify-end gap-3">
            <button @click="showExportModal = false" class="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium transition-colors">Cancel</button>
            <button @click="submitExport" class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg shadow-indigo-500/20 transition-colors">Download</button>
        </div>
      </div>
    </div>



    <!-- Digital Contact Details Modal -->
    <div v-if="showDigitalModal && viewingDigitalContact" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div class="bg-slate-900 border border-slate-700 rounded-2xl shadow-2xl w-full max-w-lg overflow-hidden flex flex-col max-h-[85vh]">
         <div class="p-6 border-b border-slate-800 flex justify-between items-center shrink-0">
            <h2 class="text-xl font-bold text-white">Contact Details</h2>
            <button @click="closeDigitalModal" class="text-slate-400 hover:text-white">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
            </button>
        </div>
        <div class="p-6 space-y-4 overflow-y-auto custom-scrollbar">
            <div class="flex items-center gap-4 mb-4">
                <div class="p-4 rounded-full bg-slate-800 border border-slate-700">
                     <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-indigo-400"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>
                </div>
                <div>
                    <div class="text-2xl font-bold text-white">{{ viewingDigitalContact.Name }}</div>
                    <div class="text-indigo-400 font-mono">{{ viewingDigitalContact.Callsign }}</div>
                </div>
            </div>

            <div class="grid grid-cols-2 gap-4 text-sm">
                <div>
                    <label class="block text-slate-500 mb-1">DMR ID</label>
                    <div class="text-white font-mono bg-slate-950 p-2 rounded border border-slate-800 select-all">{{ viewingDigitalContact.DMRID }}</div>
                </div>
                 <div>
                    <label class="block text-slate-500 mb-1">Type</label>
                    <div class="text-white p-2">{{ viewingDigitalContact.Type }}</div>
                </div>
                 <div class="col-span-2">
                    <label class="block text-slate-500 mb-1">Location</label>
                    <div class="text-white p-2 border border-slate-800 rounded bg-slate-950/50">
                        {{ viewingDigitalContact.City || 'N/A' }}, {{ viewingDigitalContact.State || 'N/A' }}, {{ viewingDigitalContact.Country || 'N/A' }}
                    </div>
                </div>
                 <div class="col-span-2" v-if="viewingDigitalContact.Remarks">
                    <label class="block text-slate-500 mb-1">Remarks</label>
                    <div class="text-slate-300 p-2 border border-slate-800 rounded bg-slate-950/50 italic">
                        {{ viewingDigitalContact.Remarks }}
                    </div>
                </div>
            </div>

            <!-- Raw Data Toggle -->
             <div class="border-t border-slate-800 pt-4 mt-2">
                <button @click="showRawData = !showRawData" class="flex items-center gap-2 text-sm text-indigo-400 hover:text-indigo-300 transition-colors w-full">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" :class="{ 'rotate-90': showRawData }" class="transition-transform"><path d="m9 18 6-6-6-6"/></svg>
                    <span>{{ showRawData ? 'Hide' : 'Show' }} Raw JSON Data</span>
                </button>

                <div v-if="showRawData" class="mt-4 bg-slate-950 rounded-lg border border-slate-800 p-4 font-mono text-xs overflow-x-auto">
                    <table class="w-full text-left">
                        <tbody>
                            <tr v-for="(value, key) in viewingDigitalContact" :key="key" class="border-b border-slate-800/50 last:border-0 hover:bg-slate-900/50">
                                <td class="py-2 pr-4 text-slate-500 font-semibold select-none">{{ key }}</td>
                                <td class="py-2 text-slate-300 break-all select-all">{{ value }}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
        <div class="p-6 border-t border-slate-800 flex justify-end gap-3">
            <button @click="closeDigitalModal" class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg shadow-indigo-500/20 transition-colors">Close</button>
        </div>
      </div>
    </div>

    <!-- Zone Modal -->
    <div v-if="showZoneModal && editingZone" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div class="bg-slate-900 border border-slate-700 rounded-2xl shadow-2xl w-full max-w-4xl overflow-hidden flex flex-col max-h-[90vh]">
        <div class="p-6 border-b border-slate-800 flex justify-between items-center">
          <h2 class="text-xl font-bold text-white">{{ editingZone.ID === 0 ? 'Create Zone' : 'Edit Zone' }}</h2>
          <button @click="closeZoneModal" class="text-slate-400 hover:text-white">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
          </button>
        </div>
        <div class="p-6 flex-1 overflow-y-auto">
            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-400 mb-1">Zone Name</label>
                <input 
                  v-model="editingZone.Name"
                  type="text" 
                  class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-lg focus:outline-none focus:border-indigo-500 text-white"
                >
            </div>
            
            <div class="mb-2 text-sm text-slate-400">Manage Channels (Drag and Drop / Buttons)</div>
            
            <ZoneEditor v-model="editingZone" :allChannels="channels" />

        </div>
        <div class="p-6 border-t border-slate-800 flex justify-end gap-3">
          <button @click="closeZoneModal" class="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium transition-colors">Cancel</button>
          <button @click="saveZone" class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-lg shadow-indigo-500/20 transition-colors">Save Zone</button>
        </div>
      </div>
    </div>

  </div>
</template>

<style>
/* Custom scrollbar for webkit */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}
::-webkit-scrollbar-track {
  background: #0f172a; 
}
::-webkit-scrollbar-thumb {
  background: #334155; 
  border-radius: 4px;
}
::-webkit-scrollbar-thumb:hover {
  background: #475569; 
}
</style>
