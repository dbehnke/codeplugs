<template>
  <div class="h-screen flex flex-col">
    <!-- Header -->
    <div class="h-16 border-b border-slate-800 flex items-center justify-between px-6 bg-slate-900/50 backdrop-blur-sm sticky top-0 z-20">
       <div class="flex items-center gap-3">
          <button v-if="selectedList" @click="selectedList = null" class="p-1 rounded hover:bg-slate-800 text-slate-400">
             <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>
          </button>
          <div>
            <h1 class="text-lg font-bold text-white flex items-center gap-2">
                <span class="text-indigo-400">Filter Lists</span>
                <span v-if="selectedList" class="text-slate-500">/ {{ selectedList.Name }}</span>
            </h1>
            <p v-if="!selectedList" class="text-xs text-slate-500">Manage custom contact filter lists.</p>
          </div>
       </div>

       <!-- Search (Only in Detail View) -->
       <div v-if="selectedList" class="relative">
          <input 
            v-model="searchQuery" 
            @input="handleSearch"
            type="text" 
            placeholder="Search ID..." 
            class="bg-slate-800 border-none rounded-full py-1.5 pl-9 pr-4 text-sm text-slate-300 focus:ring-1 focus:ring-indigo-500 w-64 placeholder-slate-500"
          >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 absolute left-3 top-2 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-6 relative">
        <!-- List of Lists -->
        <div v-if="!selectedList" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             <div v-if="loading" class="col-span-full flex justify-center py-10">
                <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
            </div>
            
            <div v-else-if="lists.length === 0" class="col-span-full text-center py-20 text-slate-500">
                No filter lists found. Import one using the Import tool.
            </div>

            <div v-for="list in lists" :key="list.ID" 
                 class="bg-slate-800 border border-slate-700 rounded-lg p-5 hover:border-indigo-500/50 transition-colors cursor-pointer group relative"
                 @click="viewList(list)">
                
                <h3 class="font-bold text-white text-lg mb-1">{{ list.Name }}</h3>
                <p class="text-sm text-slate-400 mb-4">{{ list.Description }}</p>
                
                <div class="flex items-center justify-between mt-auto">
                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-700 text-slate-300">
                        {{ list.Count }} entries
                    </span>
                    <button @click.stop="deleteList(list)" class="text-slate-500 hover:text-red-400 p-1 opacity-0 group-hover:opacity-100 transition-opacity" title="Delete List">
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>
                    </button>
                </div>
            </div>
        </div>

        <!-- Detail View (Entries) -->
        <div v-else>
            <div class="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden">
                <div class="overflow-x-auto">
                    <table class="w-full text-left text-sm text-slate-400">
                        <thead class="bg-slate-800 text-slate-200 uppercase font-bold text-xs">
                            <tr>
                                <th class="px-6 py-3">DMR ID</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-slate-800">
                             <tr v-if="loadingEntries">
                                <td class="px-6 py-4 text-center">Loading...</td>
                            </tr>
                            <tr v-else-if="entries.length === 0">
                                <td class="px-6 py-4 text-center">No entries found.</td>
                            </tr>
                            <tr v-for="entry in entries" :key="entry.ID" class="hover:bg-slate-800/50">
                                <td class="px-6 py-3 font-mono text-indigo-300">{{ entry.DMRID }}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                 <!-- Pagination -->
                <div class="bg-slate-800 px-6 py-3 border-t border-slate-700 flex items-center justify-between">
                    <span class="text-xs text-slate-400">
                        Page {{ meta.page }} of {{ Math.ceil(meta.total / meta.limit) }} ({{ meta.total }} items)
                    </span>
                    <div class="flex gap-2">
                        <button 
                            @click="changePage(meta.page - 1)" 
                            :disabled="meta.page <= 1"
                            class="px-2 py-1 rounded bg-slate-700 text-slate-300 hover:text-white disabled:opacity-50 text-xs">
                            Prev
                        </button>
                        <button 
                            @click="changePage(meta.page + 1)" 
                            :disabled="meta.page * meta.limit >= meta.total"
                            class="px-2 py-1 rounded bg-slate-700 text-slate-300 hover:text-white disabled:opacity-50 text-xs">
                            Next
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'

interface ListSummary {
    ID: number
    Name: string
    Description: string
    Count: number
}

interface Entry {
    ID: number
    DMRID: number
}

const lists = ref<ListSummary[]>([])
const selectedList = ref<ListSummary | null>(null)
const entries = ref<Entry[]>([])
const loading = ref(false)
const loadingEntries = ref(false)

const searchQuery = ref('')
const meta = ref({ page: 1, limit: 100, total: 0 })

const fetchLists = async () => {
    loading.value = true
    try {
        const res = await fetch('/api/filter_lists')
        lists.value = await res.json() || []
    } catch (e) {
        console.error("Error fetching lists", e)
    } finally {
        loading.value = false
    }
}

const fetchEntries = async () => {
    if (!selectedList.value) return
    loadingEntries.value = true
    try {
        const params = new URLSearchParams({
            id: selectedList.value.ID.toString(),
            page: meta.value.page.toString(),
            limit: meta.value.limit.toString(),
            search: searchQuery.value
        })
        const res = await fetch(`/api/filter_lists?${params}`)
        const data = await res.json()
        entries.value = data.entries || []
        meta.value = data.meta
    } catch (e) {
         console.error("Error fetching entries", e)
    } finally {
        loadingEntries.value = false
    }
}

const viewList = (list: ListSummary) => {
    selectedList.value = list
    meta.value.page = 1
    searchQuery.value = ''
    fetchEntries()
}

const changePage = (newPage: number) => {
    if (newPage < 1) return
    meta.value.page = newPage
    fetchEntries()
}

let searchTimeout: any
const handleSearch = () => {
    clearTimeout(searchTimeout)
    searchTimeout = setTimeout(() => {
        meta.value.page = 1
        fetchEntries()
    }, 300)
}

const deleteList = async (list: ListSummary) => {
    if (!confirm(`Are you sure you want to delete list "${list.Name}"?`)) return
    try {
        await fetch(`/api/filter_lists?id=${list.ID}`, { method: 'DELETE' })
        await fetchLists()
    } catch (e) {
        alert("Failed to delete list")
    }
}

onMounted(() => {
    fetchLists()
})
</script>
