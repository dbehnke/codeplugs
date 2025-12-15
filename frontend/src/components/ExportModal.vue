<template>
  <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
    <div class="bg-slate-800 border border-slate-700 rounded-lg shadow-xl w-full max-w-md p-6 relative">
      <button @click="$emit('close')" class="absolute top-4 right-4 text-slate-400 hover:text-white">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
      </button>
      
      <h2 class="text-xl font-bold text-white mb-4">Export Codeplug</h2>

      <!-- Format Selection -->
      <div class="mb-4">
        <label class="block text-sm font-medium text-slate-400 mb-1">Export Format</label>
        <select v-model="selectedFormat" class="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-white focus:outline-none focus:border-indigo-500">
          <option value="at890">AnyTone 890 (Zip)</option>
          <option value="dm32uv">Baofeng DM32UV (Zip)</option>
          <option value="chirp">CHIRP / Generic (CSV)</option>
          <option value="db">Database Backup (.db)</option>
        </select>
        <p class="text-xs text-slate-500 mt-1">
          <span v-if="selectedFormat === 'at890'">Full export including all zones, channels, and contacts.</span>
          <span v-if="selectedFormat === 'dm32uv'">Full export for DM32UV radio (Zip).</span>
          <span v-if="selectedFormat === 'chirp'">Exports Channels only. Digital contacts logic may be limited.</span>
          <span v-if="selectedFormat === 'db'">Download full SQLite database file for backup.</span>
        </p>
      </div>

      <!-- Zone Selection -->
      <div class="mb-6">
        <label class="block text-sm font-medium text-slate-400 mb-2">Zones to Export</label>
        
        <div class="space-y-2 max-h-48 overflow-y-auto bg-slate-900/50 p-2 rounded border border-slate-700/50">
          <!-- All Option -->
          <label class="flex items-center space-x-2 p-1 hover:bg-slate-800 rounded cursor-pointer">
            <input type="checkbox" v-model="selectAll" class="rounded border-slate-600 bg-slate-800 text-indigo-500 focus:ring-0 focus:ring-offset-0">
            <span class="text-sm text-white">All Zones</span>
          </label>
          
          <div class="h-px bg-slate-700 my-1"></div>

          <!-- Individual Zones -->
          <label v-for="zone in zones" :key="zone.ID" class="flex items-center space-x-2 p-1 hover:bg-slate-800 rounded cursor-pointer">
            <input type="checkbox" :value="zone.ID" v-model="selectedZoneIDs" :disabled="selectAll" class="rounded border-slate-600 bg-slate-800 text-indigo-500 focus:ring-0 focus:ring-offset-0 disabled:opacity-50">
            <span class="text-sm text-slate-300" :class="{'opacity-50': selectAll}">{{ zone.Name }}</span>
          </label>
        </div>
      </div>

      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" class="px-4 py-2 rounded text-slate-300 hover:text-white hover:bg-slate-700 transition-colors">
          Cancel
        </button>
        <button @click="handleExport" class="px-4 py-2 rounded bg-indigo-600 text-white hover:bg-indigo-500 transition-colors flex items-center gap-2">
          <span v-if="isExporting">Exporting...</span>
          <span v-else>Export</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps<{
  isOpen: boolean
  zones: any[]
}>()

const emit = defineEmits(['close', 'export'])

const selectedFormat = ref('at890')
const selectAll = ref(true)
const selectedZoneIDs = ref<number[]>([])
const isExporting = ref(false)

// Watch for selectAll change to clear individual selections or sync behavior logic if needed
// Actually, if selectAll is true, we ignore selectedZoneIDs during export construction
watch(selectAll, (newVal) => {
  if (newVal) {
    selectedZoneIDs.value = []
  }
})

const handleExport = () => {
  isExporting.value = true
  
  // Construct URL params
  let url = `/api/export?format=${selectedFormat.value}`
  
  if (!selectAll.value && selectedZoneIDs.value.length > 0) {
    // Append zone IDs
    const ids = selectedZoneIDs.value.join(',')
    url += `&zone_id=${ids}`
  }

  // Trigger download via window.location (simplest for file download)
  window.location.href = url
  
  // Close modal after short delay
  setTimeout(() => {
    isExporting.value = false
    emit('close')
  }, 1000)
}
</script>
