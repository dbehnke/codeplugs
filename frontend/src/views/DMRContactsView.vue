<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useCodeplugStore, type Contact } from '../stores/codeplug'

const store = useCodeplugStore()

const page = ref(1)
const total = ref(0)
const search = ref('')
const sort = ref('name')
const order = ref('asc')
const highlightFilter = ref('')
const highlightedIDs = ref(new Set<number>())
const filterLists = ref<any[]>([])

let searchTimeout: any

const fetchContacts = async () => {
    const meta = await store.fetchDMRContacts(page.value, 50, search.value, sort.value, order.value)
    total.value = meta.total
}

const fetchFilterLists = async () => {
    try {
        const res = await fetch('/api/filter_lists')
        if (res.ok) {
            filterLists.value = await res.json()
        }
    } catch (e) {
        console.error("Failed to fetch filter lists", e)
    }
}

const updateHighlights = async () => {
    if (!highlightFilter.value) {
        highlightedIDs.value.clear()
        return
    }
    // Fetch IDs for this list
    try {
        // First find list ID
        const list = filterLists.value.find(l => l.Name === highlightFilter.value)
        if (!list) return

        const res = await fetch(`/api/filter_lists?id=${list.ID}&mode=ids`)
        if (res.ok) {
            const ids: number[] = await res.json()
            highlightedIDs.value = new Set(ids)
        }
    } catch (e) {
        console.error("Failed to fetch ids", e)
    }
}

watch(highlightFilter, () => updateHighlights())

onMounted(() => {
    fetchContacts()
    fetchFilterLists()
})

watch(page, () => fetchContacts())

const handleSearch = () => {
    clearTimeout(searchTimeout)
    searchTimeout = setTimeout(() => {
        page.value = 1
        fetchContacts()
    }, 300)
}

const toggleSort = (field: string) => {
    if (sort.value === field) {
        order.value = order.value === 'asc' ? 'desc' : 'asc'
    } else {
        sort.value = field
        order.value = 'asc'
    }
    fetchContacts()
}
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Toolbar -->
    <div class="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-900/50 backdrop-blur-sm sticky top-0 z-10">
      <div class="flex items-center gap-4 flex-1">
         <h1 class="text-xl font-bold text-slate-100 whitespace-nowrap">DMR CSV Contacts</h1>
         <input 
              v-model="search"
              @input="handleSearch"
              type="text" 
              placeholder="Search by Name, Callsign, or ID..." 
              class="max-w-md w-full px-4 py-2 bg-slate-950/50 border border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
        >
        
        <!-- Highlight Dropdown -->
        <select v-model="highlightFilter" class="bg-slate-950/50 border border-slate-700 rounded-xl px-3 py-2 text-sm text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/50">
            <option value="">No Highlight</option>
            <option v-for="list in filterLists" :key="list.ID" :value="list.Name">
                Highlight: {{ list.Name }}
            </option>
        </select>
      </div>

       <div class="flex items-center gap-4">
           <span v-if="store.loadingDMRContacts" class="text-sm text-slate-400">Loading...</span>
           <span class="text-sm text-slate-400">
               Page {{ page }} of {{ Math.ceil(total / 50) }}
           </span>
           <div class="flex gap-1">
               <button @click="page--" :disabled="page <= 1" class="px-3 py-1 rounded bg-slate-800 text-slate-300 disabled:opacity-50 hover:bg-slate-700">Prev</button>
               <button @click="page++" :disabled="page >= Math.ceil(total / 50)" class="px-3 py-1 rounded bg-slate-800 text-slate-300 disabled:opacity-50 hover:bg-slate-700">Next</button>
           </div>
       </div>
    </div>

    <!-- Table -->
     <div class="flex-1 overflow-auto">
      <table class="w-full text-left border-collapse">
        <thead class="sticky top-0 bg-slate-900 z-10 shadow-sm">
          <tr class="text-xs uppercase tracking-wider text-slate-500 font-semibold cursor-pointer select-none">
            <th class="px-6 py-4 bg-slate-900 hover:text-slate-300" @click="toggleSort('dmr_id')">DMR ID 
                <span v-if="sort === 'dmr_id'">{{ order === 'asc' ? '↑' : '↓' }}</span>
            </th>
             <th class="px-6 py-4 bg-slate-900 hover:text-slate-300" @click="toggleSort('callsign')">Callsign
                 <span v-if="sort === 'callsign'">{{ order === 'asc' ? '↑' : '↓' }}</span>
             </th>
            <th class="px-6 py-4 bg-slate-900 hover:text-slate-300" @click="toggleSort('name')">Name
                <span v-if="sort === 'name'">{{ order === 'asc' ? '↑' : '↓' }}</span>
            </th>
            <th class="px-6 py-4 bg-slate-900">Details</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/50">
          <tr v-for="c in store.dmrContacts" :key="c.ID" 
              class="group hover:bg-slate-800/30 transition-colors"
              :class="{'bg-indigo-900/30 hover:bg-indigo-900/50': highlightedIDs.has(c.DMRID)}">
            <td class="px-6 py-3 font-mono text-indigo-300">{{ c.DMRID }}</td>
            <td class="px-6 py-3 font-medium text-slate-200">{{ c.Callsign }}</td>
            <td class="px-6 py-3 text-slate-400">{{ c.Name }}</td>
             <td class="px-6 py-3 text-slate-500 text-sm">
                 {{ c.City }} {{ c.State }} {{ c.Country }}
             </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
