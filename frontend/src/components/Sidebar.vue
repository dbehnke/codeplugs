<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { 
  Settings, 
  Radio, 
  Database,
  Menu,
  ChevronRight,
  ChevronDown,
  Upload,
  Download
} from 'lucide-vue-next'
import ImportModal from './ImportModal.vue'
import ExportModal from './ExportModal.vue'
import { useCodeplugStore } from '../stores/codeplug'

const store = useCodeplugStore()
const route = useRoute()
const collapsed = ref(false)

const showImportModal = ref(false)
const showExportModal = ref(false)

const sections = ref([
  {
    title: 'Common Setting',
    icon: Settings,
    expanded: true,
    items: [
      { name: 'Channel', path: '/channels' },
      { name: 'Zone', path: '/zones' },
      { name: 'Scan List', path: '/scanlists' },
      { name: 'Roaming Channel', path: '/roaming/channels' },
      { name: 'Roaming Zone', path: '/roaming/zones' },
    ]
  },
  {
    title: 'DMR',
    icon: Radio,
    expanded: true,
    items: [
       { name: 'Talk Groups', path: '/dmr-talkgroups' },
       { name: 'CSV Contacts', path: '/dmr-contacts' },
       { name: 'Filter Lists', path: '/filter-lists' },
       { name: 'Radio ID List', path: '#' }, // Placeholder
    ]
  },
   {
    title: 'NXDN',
    icon: Database,
    expanded: true,
    items: [
       { name: 'Talk Groups', path: '/nxdn-talkgroups' },
       { name: 'CSV Contacts', path: '/nxdn-contacts' },
    ]
  },
])

const toggleSection = (index: number) => {
    sections.value[index].expanded = !sections.value[index].expanded
}

// Refresh data on import success
const handleImportSuccess = async () => {
    await store.fetchChannels()
    await store.fetchZones()
}

onMounted(async () => {
    // Ensure zones are loaded for export modal
    if (store.zones.length === 0) {
        await store.fetchZones()
    }
})
</script>

<template>
  <div class="h-screen bg-slate-950 border-r border-slate-800 flex flex-col transition-all duration-300" 
       :class="collapsed ? 'w-16' : 'w-64'">
    
    <!-- Branding -->
    <div class="p-4 border-b border-slate-800 flex items-center justify-between">
      <div v-if="!collapsed" class="font-bold text-indigo-400 text-lg tracking-tight truncate">Universal Codeplug</div>
      <button @click="collapsed = !collapsed" class="p-1 rounded hover:bg-slate-800 text-slate-400">
          <Menu class="w-5 h-5" />
      </button>
    </div>

    <!-- Navigation -->
    <div class="flex-1 overflow-y-auto py-2">
      <div v-for="(section, idx) in sections" :key="idx" class="mb-2">
         <!-- Section Header -->
         <div v-if="!collapsed" 
              @click="toggleSection(idx)"
              class="px-4 py-2 flex items-center justify-between cursor-pointer hover:bg-slate-900 text-slate-300 text-sm font-semibold select-none">
             <div class="flex items-center gap-2">
                <component :is="section.icon" class="w-4 h-4 text-indigo-500" />
                <span>{{ section.title }}</span>
             </div>
             <component :is="section.expanded ? ChevronDown : ChevronRight" class="w-4 h-4 text-slate-600" />
         </div>
         <div v-else class="flex justify-center py-2" :title="section.title">
             <component :is="section.icon" class="w-5 h-5 text-indigo-500" />
         </div>

         <!-- Items -->
         <div v-if="!collapsed && section.expanded" class="mt-1">
             <router-link 
                v-for="item in section.items" 
                :key="item.name"
                :to="item.path"
                class="block pl-10 pr-4 py-1.5 text-sm text-slate-400 hover:text-white hover:bg-slate-900 border-l-2 border-transparent transition-colors"
                :class="{ 'border-indigo-500 bg-slate-900 text-white': route.path === item.path, 'opacity-50 cursor-not-allowed': item.path === '#' }"
             >
                {{ item.name }}
             </router-link>
         </div>
      </div>
    </div>

    <!-- Data Actions -->
    <div class="p-2 border-t border-slate-800 bg-slate-900/30 flex gap-2 justify-center">
        <button @click="showImportModal = true" class="flex-1 flex items-center justify-center gap-2 p-2 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm transition-colors" title="Import">
            <Upload class="w-4 h-4" />
            <span v-if="!collapsed">Import</span>
        </button>
        <button @click="showExportModal = true" class="flex-1 flex items-center justify-center gap-2 p-2 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm transition-colors" title="Export">
            <Download class="w-4 h-4" />
            <span v-if="!collapsed">Export</span>
        </button>
    </div>

    <!-- Device Status / Footer -->
    <div class="p-4 border-t border-slate-800 bg-slate-900/50">
        <div v-if="!collapsed" class="text-xs text-slate-500 flex justify-between">
            <span>Model: Universal</span>
            <span class="text-emerald-500">Connected</span>
        </div>
        <div v-else class="w-2 h-2 rounded-full bg-emerald-500 mx-auto"></div>
    </div>

    <!-- Modals -->
    <ImportModal :is-open="showImportModal" @close="showImportModal = false" @import-success="handleImportSuccess" />
    <ExportModal :is-open="showExportModal" :zones="store.zones" @close="showExportModal = false" />

  </div>
</template>
