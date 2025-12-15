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
          <option value="generic">Generic / DB25-D (CSV)</option>
          <option value="zip">Zip / Folder Import</option>
          <option value="radioid">RadioID / Digital Contacts</option>
          <option value="db">Database Restore (.db)</option>
        </select>
         <p class="text-xs text-slate-500 mt-1" v-if="selectedFormat==='zip'">
          Supports Zip archives from DM32UV, AnyTone 890, or backups.
        </p>
         <p class="text-xs text-slate-500 mt-1" v-if="selectedFormat==='db'">
          <span class="text-amber-500">WARNING:</span> This will replace the entire database.
        </p>
      </div>
      
       <!-- File Input -->
      <div class="mb-6">
        <label class="block text-sm font-medium text-slate-400 mb-1">Select File</label>
        <div class="flex items-center justify-center w-full">
            <label for="dropzone-file" class="flex flex-col items-center justify-center w-full h-32 border-2 border-slate-600 border-dashed rounded-lg cursor-pointer bg-slate-700/50 hover:bg-slate-700 transition-colors">
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
                <input id="dropzone-file" type="file" class="hidden" @change="handleFileSelect" />
            </label>
        </div> 
      </div>

      <!-- Mode Toggle -->
      <div class="mb-6 p-4 bg-slate-900/50 rounded border border-slate-700/50">
        <label class="block text-sm font-medium text-slate-400 mb-2">Import Mode</label>
        <div class="flex gap-4">
             <label class="flex items-center space-x-2 cursor-pointer">
                <input type="radio" v-model="importMode" value="append" class="text-indigo-600 focus:ring-indigo-500 bg-slate-800 border-slate-600">
                <span class="text-white text-sm">Append (Default)</span>
             </label>
             <label class="flex items-center space-x-2 cursor-pointer">
                <input type="radio" v-model="importMode" value="overwrite" class="text-red-600 focus:ring-red-500 bg-slate-800 border-slate-600">
                <span class="text-white text-sm">Overwrite (Clear All)</span>
             </label>
        </div>
        <p class="text-xs text-slate-500 mt-2">
            <span v-if="importMode === 'append'">Adds new channels. Skips duplicates (Name + Freq).</span>
            <span v-else class="text-amber-500">WARNING: Deletes ALL existing data of the imported type before importing.</span>
        </p>
      </div>

      <!-- Progress / Error -->
      <div v-if="uploadStatus" class="mb-4 text-sm" :class="{'text-green-400': uploadStatus.type === 'success', 'text-red-400': uploadStatus.type === 'error', 'text-blue-400': uploadStatus.type === 'info'}">
          {{ uploadStatus.message }}
      </div>

      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" class="px-4 py-2 rounded text-slate-300 hover:text-white hover:bg-slate-700 transition-colors">
          Cancel
        </button>
        <button @click="handleImport" :disabled="!selectedFile || isUploading" class="px-4 py-2 rounded bg-indigo-600 text-white hover:bg-indigo-500 transition-colors disabled:opacity-50 flex items-center gap-2">
          <span v-if="isUploading">Importing...</span>
          <span v-else>Import</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  isOpen: boolean
}>()

const emit = defineEmits(['close', 'import-success'])

const selectedFormat = ref('generic')
const importMode = ref('append')
const selectedFile = ref<File | null>(null)
const isUploading = ref(false)
const uploadStatus = ref<{type: string, message: string} | null>(null)

const handleFileSelect = (event: Event) => {
    const target = event.target as HTMLInputElement
    if (target.files && target.files.length > 0) {
        selectedFile.value = target.files[0]
        
        // Auto-detect zip
        if (selectedFile.value.name.endsWith('.zip')) {
            selectedFormat.value = 'zip'
        }
    }
}

const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

const handleImport = async () => {
    if (!selectedFile.value) return

    isUploading.value = true
    uploadStatus.value = { type: 'info', message: 'Uploading...' }

    const formData = new FormData()
    formData.append('file', selectedFile.value)
    formData.append('format', selectedFormat.value)
    
    if (importMode.value === 'overwrite') {
        formData.append('overwrite', 'true')
    }

    try {
        const response = await fetch('/api/import', {
            method: 'POST',
            body: formData
        })

        if (!response.ok) {
            const text = await response.text()
            throw new Error(text || 'Import failed')
        }

        const result = await response.json().catch(() => ({ message: 'Import successful' })) // Handle plain text response fallback
        
        uploadStatus.value = { 
            type: 'success', 
            message: result.message || `Successfully imported file: ${selectedFile.value.name}` 
        }

        setTimeout(() => {
            emit('import-success')
            emit('close')
            // Reset state
            selectedFile.value = null
            uploadStatus.value = null
            isUploading.value = false
        }, 1500)

    } catch (e: any) {
        uploadStatus.value = { type: 'error', message: e.message }
        isUploading.value = false
    }
}
</script>
