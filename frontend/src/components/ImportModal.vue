<template>
  <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
    <div class="bg-slate-800 border border-slate-700 rounded-lg shadow-xl w-full max-w-md p-6 relative">
      <button @click="$emit('close')" class="absolute top-4 right-4 text-slate-400 hover:text-white">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
      </button>
      
      <h2 class="text-xl font-bold text-white mb-4">Import Codeplug</h2>

      <!-- Format Selection -->
      <div class="mb-4">
        <label class="block text-sm font-medium text-slate-400 mb-1">Import Format</label>
        <select v-model="selectedFormat" class="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-white focus:outline-none focus:border-indigo-500">
          <option value="generic">Generic / Chirp (CSV)</option>
          <option value="zip">Zip / Folder Import</option>
          <option value="radioid">RadioID / Digital Contacts</option>
          <option value="filter_list">Contact Filter List</option>
          <option value="db">Database Restore (.db)</option>
        </select>
         <p class="text-xs text-slate-500 mt-1" v-if="selectedFormat==='zip'">
          Supports Zip archives from DM32UV, AnyTone 890, or backups.
        </p>
         <p class="text-xs text-slate-500 mt-1" v-if="selectedFormat==='db'">
          <span class="text-amber-500">WARNING:</span> This will replace the entire database.
        </p>
      </div>

      <!-- List Name Input (for Filter List) -->
      <div class="mb-4" v-if="selectedFormat === 'filter_list'">
        <label class="block text-sm font-medium text-slate-400 mb-1">List Name <span class="text-red-500">*</span></label>
        <input v-model="listName" type="text" placeholder="e.g. brandmeister_active" class="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-white focus:outline-none focus:border-indigo-500" />
        <p class="text-xs text-slate-500 mt-1">If a list with this name exists, it will be overwritten.</p>
      </div>

      <!-- Import Logic (RadioID Source) -->
      <div v-if="selectedFormat === 'radioid'">
          <div class="mb-4">
              <label class="block text-sm font-medium text-slate-400 mb-2">Source</label>
              <div class="flex gap-4">
                  <label class="flex items-center gap-2 cursor-pointer">
                      <input type="radio" v-model="sourceMode" value="upload" class="bg-slate-800 border-slate-600 text-indigo-600 focus:ring-indigo-500">
                      <span class="text-slate-300">File Upload</span>
                  </label>
                  <label class="flex items-center gap-2 cursor-pointer">
                      <input type="radio" v-model="sourceMode" value="download" class="bg-slate-800 border-slate-600 text-indigo-600 focus:ring-indigo-500">
                      <span class="text-slate-300">Download from RadioID.net</span>
                  </label>
              </div>
          </div>
      </div>
      
       <!-- File Input -->
      <div class="mb-6" v-if="sourceMode === 'upload' && selectedFormat !== 'db_restore_no_file_needed_Wait_db_needs_file'">
        <!-- DB restore needs file, Generic needs file, Zip needs file. RadioID download NO file. -->
        <label class="block text-sm font-medium text-slate-400 mb-1">Select File</label>
        <div class="flex items-center justify-center w-full">
            <label class="flex flex-col items-center justify-center w-full h-32 border-2 border-slate-600 border-dashed rounded-lg cursor-pointer bg-slate-700/50 hover:bg-slate-700 transition-colors"
                @drop.prevent="handleDrop"
                @dragover.prevent>
                <div class="flex flex-col items-center justify-center pt-5 pb-6" v-if="!selectedFile">
                    <svg class="w-8 h-8 mb-4 text-slate-400" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 20 16">
                        <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 13h3a3 3 0 0 0 0-6h-.025A5.56 5.56 0 0 0 16 6.5 5.5 5.5 0 0 0 5.207 5.021C5.137 5.017 5.071 5 5 5a4 4 0 0 0 0 8h2.167M10 15V6m0 0L8 8m2-2 2 2"/>
                    </svg>
                    <p class="text-sm text-slate-400"><span class="font-semibold">Click to upload</span> or drag and drop</p>
                    <p class="text-xs text-slate-500">CSV or ZIP files</p>
                </div>
                <div v-else class="flex flex-col items-center justify-center pt-5 pb-6">
                    <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-green-500 mb-2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                    <p class="text-sm text-white font-medium">{{ selectedFile.name }}</p>
                    <p class="text-xs text-slate-400">{{ formatSize(selectedFile.size) }}</p>
                </div>
                <input type="file" ref="fileInput" class="hidden" @change="handleFileChange" :accept="acceptTypes" />
            </label>
        </div> 
      </div>

      <!-- Mode Toggle for Generic (Overwrite/Append) -->
       <div class="mb-6 p-4 bg-slate-900/50 rounded border border-slate-700/50" v-if="selectedFormat === 'generic' || selectedFormat === 'radioid'">
        <label class="block text-sm font-medium text-slate-400 mb-2">Import Mode</label>
        <div class="flex gap-4">
             <label class="flex items-center space-x-2 cursor-pointer">
                <input type="radio" v-model="overwrite" :value="false" class="text-indigo-600 focus:ring-indigo-500 bg-slate-800 border-slate-600">
                <span class="text-white text-sm">Append (Default)</span>
             </label>
             <label class="flex items-center space-x-2 cursor-pointer">
                <input type="radio" v-model="overwrite" :value="true" class="text-red-600 focus:ring-red-500 bg-slate-800 border-slate-600">
                <span class="text-white text-sm">Overwrite (Clear All)</span>
             </label>
        </div>
        <p class="text-xs text-slate-500 mt-2">
            <span v-if="!overwrite">Adds new entries. Skips duplicates.</span>
            <span v-else class="text-amber-500">WARNING: Deletes ALL existing data of the imported type before importing.</span>
        </p>
      </div>

      <!-- Progress Bar (WebSocket) -->
      <div v-if="isImporting || progress.status === 'running' || progress.status === 'completed'" class="mb-6 bg-slate-900/50 p-3 rounded border border-slate-700">
          <div class="flex justify-between text-xs text-slate-400 mb-1">
              <span>{{ progress.message || 'Processing...' }}</span>
              <span v-if="progress.total > 0">{{ Math.round((progress.processed / progress.total) * 100) }}%</span>
          </div>
          <div class="w-full bg-slate-700 rounded-full h-2.5 overflow-hidden">
              <div class="bg-indigo-600 h-2.5 rounded-full transition-all duration-300" 
                   :style="{ width: progress.total > 0 ? (progress.processed / progress.total * 100) + '%' : '0%' }"></div>
          </div>
          <div class="text-xs text-slate-500 mt-1 text-right" v-if="progress.total > 0">
              {{ progress.processed }} / {{ progress.total }} records
          </div>
      </div>

      <!-- Status Message (Result) -->
      <div v-if="uploadStatus" class="mb-4 text-sm" :class="{'text-green-400': uploadStatus.type === 'success', 'text-red-400': uploadStatus.type === 'error', 'text-blue-400': uploadStatus.type === 'info'}">
          {{ uploadStatus.message }}
      </div>

      <div class="flex justify-end gap-3 mt-6">
        <button @click="$emit('close')" class="px-4 py-2 rounded text-slate-300 hover:text-white hover:bg-slate-700 transition-colors">
          Cancel
        </button>
        <button @click="handleImport" class="px-4 py-2 rounded bg-indigo-600 text-white hover:bg-indigo-500 transition-colors flex items-center gap-2" :disabled="isImporting || (sourceMode === 'upload' && !selectedFile)">
          <span v-if="isImporting">Importing...</span>
          <span v-else>Import</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  isOpen: boolean
}>()

const emit = defineEmits(['close', 'import-complete'])

const selectedFormat = ref('generic')
const selectedFile = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const isImporting = ref(false)
const overwrite = ref(false)
const listName = ref('')
const sourceMode = ref('upload') 
const uploadStatus = ref<{type: string, message: string} | null>(null)

// Progress State
const progress = ref({
    total: 0,
    processed: 0,
    status: '',
    message: ''
})
let socket: WebSocket | null = null

const acceptTypes = computed(() => {
  if (selectedFormat.value === 'zip' || selectedFormat.value === 'dm32uv' || selectedFormat.value === 'at890') return '.zip'
  if (selectedFormat.value === 'db') return '.db'
  return '.csv,.txt'
})

const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

const triggerFileInput = () => {
  fileInput.value?.click()
}

const handleFileChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedFile.value = target.files[0]
  }
}

const handleDrop = (e: DragEvent) => {
  if (e.dataTransfer?.files.length) {
    selectedFile.value = e.dataTransfer.files[0]
  }
}

// WebSocket Management
const connectWS = () => {
    if (socket) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/ws`
    
    socket = new WebSocket(wsUrl)
    
    socket.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data)
            // Expecting format: { total, processed, status, message }
            if (typeof data.processed === 'number') {
                progress.value = data
            }
        } catch (e) {
            // Ignore non-json messages
        }
    }

    socket.onclose = () => {
        socket = null
    }
}

const disconnectWS = () => {
    if (socket) {
        socket.close()
        socket = null
    }
}

onMounted(() => {
    if (props.isOpen) connectWS()
})

onUnmounted(() => {
    disconnectWS()
})

// Re-connect when modal opens if mounted
import { watch } from 'vue'
watch(() => props.isOpen, (newVal) => {
    if (newVal) connectWS()
    else {
        // Delay disconnect slightly to allow final status to be seen? 
        // Or just keep it open. Actually better to disconnect to save resources.
        disconnectWS()
    }
})

const handleImport = async () => {
  if (sourceMode.value === 'upload' && !selectedFile.value) return

  if (overwrite.value) {
    if (!confirm("Are you sure you want to overwrite all existing channels? This action cannot be undone.")) {
      return
    }
  }

  isImporting.value = true
  uploadStatus.value = null // Clear old status, rely on progress bar mostly
  progress.value = { total: 0, processed: 0, status: 'starting', message: 'Starting upload...' }

  const formData = new FormData()
  if (selectedFile.value) {
      formData.append('file', selectedFile.value)
  }
  formData.append('overwrite', overwrite.value.toString())
  formData.append('list_name', listName.value)
  formData.append('source_mode', sourceMode.value)

  let url = `/api/import?format=${selectedFormat.value}`

  try {
    const res = await fetch(url, {
      method: 'POST',
      body: formData
    })
    
    if (!res.ok) {
        const txt = await res.text() 
        uploadStatus.value = { type: 'error', message: 'Import failed: ' + txt }
        progress.value.status = 'error'
    } else {
        const json = await res.json()
        // Success
        progress.value.status = 'completed' 
        progress.value.processed = progress.value.total // Ensure 100%
        uploadStatus.value = { type: 'success', message: json.message || 'Import successful' }
        
        setTimeout(() => {
             emit('import-complete')
             emit('close')
             // Reset
             selectedFile.value = null
             uploadStatus.value = null
             progress.value = { total: 0, processed: 0, status: '', message: '' }
             isImporting.value = false
        }, 1500)
    }
  } catch (e: any) {
    uploadStatus.value = { type: 'error', message: 'Import error: ' + e.message }
    progress.value.status = 'error'
    isImporting.value = false
  } 
  // Note: we don't set isImporting = false immediately on success so the modal doesn't flash capable of re-importing while closing
}
</script>
