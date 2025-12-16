<template>
  <div class="flex flex-row gap-4 h-[400px]">
    <div v-if="debugInfo" class="absolute top-0 right-0 bg-red-100 p-2 z-50 text-xs">
        Debug: {{ debugInfo }}
    </div>
    <!-- Available Channels -->
    <div class="flex-1 border rounded flex flex-col">
      <div class="bg-gray-100 p-2 font-bold border-b">Available Channels</div>
      <div class="overflow-y-auto flex-1 p-2 available-list">
        <div 
          v-for="(ch, index) in availableChannels" 
          :key="ch.ID"
          class="channel-item p-1 hover:bg-gray-200 cursor-pointer select-none"
          :class="{ 'bg-blue-100': selectedAvailableIDs.has(ch.ID) }"
          @click="selectAvailable(ch, index, $event)"
        >
          {{ ch.Name }} ({{ ch.RxFrequency }})
        </div>
      </div>
    </div>

    <!-- Controls -->
    <div class="flex flex-col justify-center gap-2">
      <button class="btn-add px-3 py-1 bg-gray-200 rounded hover:bg-gray-300" @click="addToZone">&gt;&gt;</button>
      <button class="btn-remove px-3 py-1 bg-gray-200 rounded hover:bg-gray-300" @click="removeFromZone">&lt;&lt;</button>
    </div>

    <!-- Zone Members -->
    <div class="flex-1 border rounded flex flex-col">
      <div class="bg-gray-100 p-2 font-bold border-b">Zone Members</div>
      <div class="overflow-y-auto flex-1 p-2 member-list">
         <div 
          v-for="(ch, index) in localZone.Channels" 
          :key="ch.ID"
          class="channel-item p-1 hover:bg-gray-200 cursor-pointer flex justify-between select-none"
          :class="{ 'bg-blue-100': selectedMemberIndices.has(index) }"
          @click="selectMember(index, $event)"
        >
          <span>{{ index + 1 }}. {{ ch.Name }}</span>
        </div>
      </div>
    </div>

    <!-- Reorder Controls -->
    <div class="flex flex-col justify-center gap-2">
      <button 
        class="btn-up px-3 py-1 bg-gray-200 rounded hover:bg-gray-300 disabled:opacity-50 disabled:cursor-not-allowed" 
        :disabled="selectedMemberIndices.size !== 1"
        @click="moveUp"
      >Up</button>
      <button 
        class="btn-down px-3 py-1 bg-gray-200 rounded hover:bg-gray-300 disabled:opacity-50 disabled:cursor-not-allowed" 
        :disabled="selectedMemberIndices.size !== 1"
        @click="moveDown"
      >Down</button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  modelValue: { type: Object, required: true },
  allChannels: { type: Array, required: true }
})

const emit = defineEmits(['update:modelValue'])

// Local copy to edit
const localZone = ref({ ...props.modelValue, Channels: [...(props.modelValue.Channels || [])] })

// Watch for external changes
watch(() => props.modelValue, (newVal) => {
  localZone.value = { ...newVal, Channels: [...(newVal.Channels || [])] }
}, { deep: true })

// Selection state
const selectedAvailableIDs = ref(new Set())
const selectedMemberIndices = ref(new Set())
const debugInfo = ref('')

let lastSelectedAvailableIndex = -1
let lastSelectedMemberIndex = -1

const availableChannels = computed(() => {
  const memberIDs = new Set(localZone.value.Channels.map(c => c.ID))
  return props.allChannels.filter(c => !memberIDs.has(c.ID))
})

// --- Available Channels Selection Logic ---

function selectAvailable(ch, index, event) {
  debugInfo.value = `Meta: ${event.metaKey}, Ctrl: ${event.ctrlKey}, Shift: ${event.shiftKey}`
  const newSet = new Set(selectedAvailableIDs.value)

  if (event.shiftKey && lastSelectedAvailableIndex !== -1) {
    // Range select
    const start = Math.min(lastSelectedAvailableIndex, index)
    const end = Math.max(lastSelectedAvailableIndex, index)
    
    // If ctrl is not held, clear others first
    if (!event.ctrlKey && !event.metaKey) {
      newSet.clear()
    }
    
    for (let i = start; i <= end; i++) {
        const item = availableChannels.value[i]
        if (item) newSet.add(item.ID)
    }
  } else if (event.ctrlKey || event.metaKey) {
    // Toggle select
    if (newSet.has(ch.ID)) {
      newSet.delete(ch.ID)
    } else {
      newSet.add(ch.ID)
    }
    lastSelectedAvailableIndex = index
  } else {
    // Single select
    newSet.clear()
    newSet.add(ch.ID)
    lastSelectedAvailableIndex = index
  }
  
  selectedAvailableIDs.value = newSet
}

function addToZone() {
  if (selectedAvailableIDs.value.size > 0) {
    const toAdd = []
    // Maintain order from available list
    availableChannels.value.forEach(ch => {
        if (selectedAvailableIDs.value.has(ch.ID)) {
            toAdd.push(ch)
        }
    })
    
    if (toAdd.length > 0) {
      localZone.value.Channels.push(...toAdd)
      selectedAvailableIDs.value = new Set()
      lastSelectedAvailableIndex = -1
      emitUpdate()
    }
  }
}

// --- Zone Members Selection Logic ---

function selectMember(index, event) {
  debugInfo.value = `Meta: ${event.metaKey}, Ctrl: ${event.ctrlKey}, Shift: ${event.shiftKey}`
  const newSet = new Set(selectedMemberIndices.value)

   if (event.shiftKey && lastSelectedMemberIndex !== -1) {
    // Range select
    const start = Math.min(lastSelectedMemberIndex, index)
    const end = Math.max(lastSelectedMemberIndex, index)
    
     // If ctrl is not held, clear others first
    if (!event.ctrlKey && !event.metaKey) {
      newSet.clear()
    }
    
    for (let i = start; i <= end; i++) {
        newSet.add(i)
    }
  } else if (event.ctrlKey || event.metaKey) {
    // Toggle select
    if (newSet.has(index)) {
        newSet.delete(index)
    } else {
        newSet.add(index)
    }
    lastSelectedMemberIndex = index
  } else {
    // Single select
    newSet.clear()
    newSet.add(index)
    lastSelectedMemberIndex = index
  }
  
  selectedMemberIndices.value = newSet
}

function removeFromZone() {
  if (selectedMemberIndices.value.size > 0) {
    // Sort indices descending to remove from end first without shifting issues
    const indicesToRemove = Array.from(selectedMemberIndices.value).sort((a, b) => b - a)
    
    indicesToRemove.forEach(idx => {
        if (idx >= 0 && idx < localZone.value.Channels.length) {
            localZone.value.Channels.splice(idx, 1)
        }
    })
    
    selectedMemberIndices.value = new Set()
    lastSelectedMemberIndex = -1
    emitUpdate()
  }
}

function moveUp() {
  // Only allow move if single item selected
  if (selectedMemberIndices.value.size !== 1) return 
  
  const idx = Array.from(selectedMemberIndices.value)[0]
  if (idx > 0) {
    const item = localZone.value.Channels[idx]
    localZone.value.Channels.splice(idx, 1)
    localZone.value.Channels.splice(idx - 1, 0, item)
    
    // Update selection to follow item
    const newSet = new Set()
    newSet.add(idx - 1)
    selectedMemberIndices.value = newSet
    lastSelectedMemberIndex = idx - 1
    
    emitUpdate()
  }
}

function moveDown() {
  // Only allow move if single item selected
  if (selectedMemberIndices.value.size !== 1) return 

  const idx = Array.from(selectedMemberIndices.value)[0]
  if (idx < localZone.value.Channels.length - 1) {
    const item = localZone.value.Channels[idx]
    localZone.value.Channels.splice(idx, 1)
    localZone.value.Channels.splice(idx + 1, 0, item)
    
    // Update selection to follow item
    const newSet = new Set()
    newSet.add(idx + 1)
    selectedMemberIndices.value = newSet
    lastSelectedMemberIndex = idx + 1
    
    emitUpdate()
  }
}

function emitUpdate() {
  emit('update:modelValue', localZone.value)
}
</script>
