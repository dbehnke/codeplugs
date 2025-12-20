<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useCodeplugStore, type RoamingChannel } from '../stores/codeplug'
import { Plus, Trash2, Save, Search } from 'lucide-vue-next'

const store = useCodeplugStore()
const searchQuery = ref('')
const selectedChannel = ref<RoamingChannel | null>(null)
const isEditing = ref(false)

onMounted(async () => {
    await store.fetchRoamingChannels()
})

const selectChannel = (ch: RoamingChannel) => {
    selectedChannel.value = JSON.parse(JSON.stringify(ch))
    isEditing.value = true
}

const createNewChannel = () => {
    selectedChannel.value = {
        ID: 0,
        Name: 'New Roaming Channel',
        RxFrequency: 440.0,
        ColorCode: 1,
        TimeSlot: 1
    }
    isEditing.value = true
}

const saveChannel = async () => {
    if (!selectedChannel.value) return
    try {
        const res = await fetch('/api/roaming/channels', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(selectedChannel.value)
        })
        if (res.ok) {
            await store.fetchRoamingChannels()
            isEditing.value = false
            selectedChannel.value = null
        }
    } catch (e) {
        console.error("Failed to save roaming channel", e)
    }
}

const deleteChannel = async (id: number) => {
    if (!confirm("Are you sure you want to delete this roaming channel?")) return
    try {
        await fetch(`/api/roaming/channels?id=${id}`, { method: 'DELETE' })
        await store.fetchRoamingChannels()
        if (selectedChannel.value?.ID === id) {
            selectedChannel.value = null
            isEditing.value = false
        }
    } catch (e) {
        console.error("Failed to delete roaming channel", e)
    }
}
</script>

<template>
    <div class="h-full flex flex-col p-6 overflow-hidden">
        <header class="flex items-center justify-between mb-8">
            <div>
                <h1 class="text-3xl font-bold bg-gradient-to-r from-white to-slate-400 bg-clip-text text-transparent">
                    Roaming Channels
                </h1>
                <p class="text-slate-500 text-sm mt-1">Manage DMR roaming frequencies and settings</p>
            </div>
            <button @click="createNewChannel" 
                    class="flex items-center gap-2 px-5 py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl font-semibold shadow-lg shadow-indigo-500/20 transition-all active:scale-95">
                <Plus :size="20" />
                New Channel
            </button>
        </header>

        <div class="flex-1 flex gap-6 min-h-0">
            <!-- Table List -->
            <div class="flex-1 bg-slate-900/50 border border-slate-800 rounded-2xl flex flex-col overflow-hidden backdrop-blur-sm">
                <div class="p-4 border-b border-slate-800 bg-slate-800/20 flex items-center gap-3">
                    <Search class="text-slate-500" :size="18" />
                    <input v-model="searchQuery" placeholder="Search channels..." 
                           class="bg-transparent border-none focus:ring-0 text-sm text-slate-200 w-full" />
                </div>
                <div class="flex-1 overflow-y-auto">
                    <table class="w-full text-left border-collapse">
                        <thead class="sticky top-0 bg-slate-900 z-10">
                            <tr class="text-xs font-bold text-slate-500 uppercase tracking-wider border-b border-slate-800">
                                <th class="px-6 py-3">Name</th>
                                <th class="px-6 py-3">Frequency</th>
                                <th class="px-6 py-3">Color Code</th>
                                <th class="px-6 py-3">Time Slot</th>
                                <th class="px-6 py-3 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-slate-800/50">
                            <tr v-for="ch in store.roamingChannels" :key="ch.ID"
                                @click="selectChannel(ch)"
                                class="group hover:bg-slate-800/30 transition-colors cursor-pointer">
                                <td class="px-6 py-4 font-medium text-slate-200">{{ ch.Name }}</td>
                                <td class="px-6 py-4 text-slate-400 font-mono">{{ ch.RxFrequency.toFixed(4) }}</td>
                                <td class="px-6 py-4">
                                    <span class="px-2 py-0.5 bg-indigo-500/10 text-indigo-400 rounded-md text-xs font-bold border border-indigo-500/20">
                                        {{ ch.ColorCode }}
                                    </span>
                                </td>
                                <td class="px-6 py-4">
                                    <span class="px-2 py-0.5 bg-fuchsia-500/10 text-fuchsia-400 rounded-md text-xs font-bold border border-fuchsia-500/20">
                                        TS {{ ch.TimeSlot }}
                                    </span>
                                </td>
                                <td class="px-6 py-4 text-right">
                                    <button @click.stop="deleteChannel(ch.ID)" 
                                            class="p-2 text-slate-500 hover:text-red-400 hover:bg-red-400/10 rounded-lg transition-all opacity-0 group-hover:opacity-100">
                                        <Trash2 :size="18" />
                                    </button>
                                </td>
                            </tr>
                            <tr v-if="store.roamingChannels.length === 0">
                                <td colspan="5" class="px-6 py-20 text-center text-slate-500 italic">
                                    No roaming channels found. Click "New Channel" to begin.
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- Editor Panel -->
            <aside v-if="isEditing && selectedChannel" 
                   class="w-80 bg-slate-900/50 border border-slate-800 rounded-2xl p-6 flex flex-col gap-6 backdrop-blur-md">
                <div class="flex items-center justify-between">
                    <h2 class="font-bold text-lg text-white">Edit Channel</h2>
                    <button @click="isEditing = false" class="text-slate-500 hover:text-white">
                        <Plus class="rotate-45" :size="24" />
                    </button>
                </div>

                <div class="space-y-4">
                    <div>
                        <label class="block text-xs font-bold text-slate-500 uppercase mb-1.5">Channel Name</label>
                        <input v-model="selectedChannel.Name" 
                               class="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 transition-all" />
                    </div>
                    <div>
                        <label class="block text-xs font-bold text-slate-500 uppercase mb-1.5">RX Frequency</label>
                        <input type="number" step="0.0001" v-model.number="selectedChannel.RxFrequency" 
                               class="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2 text-white font-mono focus:ring-2 focus:ring-indigo-500/50 transition-all" />
                    </div>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="block text-xs font-bold text-slate-500 uppercase mb-1.5">Color Code</label>
                            <input type="number" v-model.number="selectedChannel.ColorCode" 
                                   class="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 transition-all" />
                        </div>
                        <div>
                            <label class="block text-xs font-bold text-slate-500 uppercase mb-1.5">Time Slot</label>
                            <select v-model.number="selectedChannel.TimeSlot" 
                                    class="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2 text-white focus:ring-2 focus:ring-indigo-500/50 transition-all">
                                <option :value="1">Slot 1</option>
                                <option :value="2">Slot 2</option>
                            </select>
                        </div>
                    </div>
                </div>

                <div class="mt-auto pt-6 flex flex-col gap-3">
                    <button @click="saveChannel" 
                            class="flex items-center justify-center gap-2 px-4 py-3 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-bold transition-all shadow-lg shadow-emerald-900/20">
                        <Save :size="18" />
                        Save Changes
                    </button>
                    <button @click="isEditing = false" 
                            class="px-4 py-3 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl font-bold transition-all">
                        Cancel
                    </button>
                </div>
            </aside>
        </div>
    </div>
</template>
