<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useCodeplugStore, type Contact } from '../stores/codeplug'

const store = useCodeplugStore()
const showModal = ref(false)
const modalContact = ref<Contact | null>(null)

onMounted(() => {
  store.fetchTalkgroups()
})

const openAddModal = () => {
    modalContact.value = {
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
    showModal.value = true
}

const openEditModal = (c: Contact) => {
    modalContact.value = { ...c }
    showModal.value = true
}

const saveContact = async () => {
    if (!modalContact.value) return
    if (!modalContact.value.Name || modalContact.value.DMRID <= 0) {
        alert("Name and valid DMR ID are required")
        return
    }

    try {
        const res = await fetch('/api/contacts', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(modalContact.value)
        })
        if (res.ok) {
            await store.fetchTalkgroups()
            showModal.value = false
        } else {
            alert("Failed to save")
        }
    } catch (e) {
        console.error(e)
    }
}

const deleteContact = async (id: number) => {
    if (confirm("Delete contact?")) {
        try {
            const res = await fetch(`/api/contacts?id=${id}`, { method: 'DELETE' })
            if (res.ok) store.fetchTalkgroups()
        } catch (e) {
            console.error(e)
        }
    }
}
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Toolbar -->
    <div class="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-900/50 backdrop-blur-sm sticky top-0 z-10">
      <h1 class="text-xl font-bold text-slate-100">DMR Talk Groups</h1>
      <button @click="openAddModal" class="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-sm font-medium shadow-lg transition-all">
          Add Talk Group
      </button>
    </div>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <table class="w-full text-left border-collapse">
        <thead class="sticky top-0 bg-slate-900 z-10 shadow-sm">
          <tr class="text-xs uppercase tracking-wider text-slate-500 font-semibold">
            <th class="px-6 py-4 bg-slate-900">Name</th>
            <th class="px-6 py-4 bg-slate-900">Type</th>
            <th class="px-6 py-4 bg-slate-900">DMR ID</th>
            <th class="px-6 py-4 bg-slate-900 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/50">
          <tr v-for="c in store.talkgroups" :key="c.ID" 
              class="group hover:bg-slate-800/30 transition-colors cursor-pointer"
              @click="openEditModal(c)">
            <td class="px-6 py-4 font-medium text-slate-200">{{ c.Name }}</td>
            <td class="px-6 py-4 text-slate-400">{{ c.Type }}</td>
            <td class="px-6 py-4 font-mono text-indigo-300">{{ c.DMRID }}</td>
            <td class="px-6 py-4 text-right">
                <button @click.stop="deleteContact(c.ID)" class="p-2 hover:bg-red-900/30 rounded-lg text-slate-400 hover:text-red-400 transition-colors">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Edit Modal -->
    <div v-if="showModal && modalContact" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
        <div class="bg-slate-900 border border-slate-700 rounded-2xl p-6 w-full max-w-md shadow-2xl">
            <h2 class="text-xl font-bold mb-4">{{ modalContact.ID === 0 ? 'Add Talk Group' : 'Edit Talk Group' }}</h2>
            <div class="space-y-4">
                <div>
                    <label class="block text-xs font-medium text-slate-400 mb-1">Name</label>
                    <input v-model="modalContact.Name" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/50" />
                </div>
                <div>
                     <label class="block text-xs font-medium text-slate-400 mb-1">Type</label>
                     <select v-model="modalContact.Type" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/50">
                         <option value="Group">Group Call</option>
                         <option value="Private">Private Call</option>
                         <option value="All Call">All Call</option>
                     </select>
                </div>
                <div>
                    <label class="block text-xs font-medium text-slate-400 mb-1">DMR ID</label>
                    <input v-model.number="modalContact.DMRID" type="number" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/50" />
                </div>
            </div>
             <div class="mt-6 flex justify-end gap-3">
                <button @click="showModal = false" class="px-4 py-2 rounded-lg bg-slate-800 text-slate-300 hover:bg-slate-700">Cancel</button>
                <button @click="saveContact" class="px-4 py-2 rounded-lg bg-indigo-600 text-white hover:bg-indigo-500">Save</button>
            </div>
        </div>
    </div>
  </div>
</template>
