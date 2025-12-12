<template>
  <div class="flex flex-row gap-4 h-[400px]">
    <!-- Available Channels -->
    <div class="flex-1 border rounded flex flex-col">
      <div class="bg-gray-100 p-2 font-bold border-b">Available Channels</div>
      <div class="overflow-y-auto flex-1 p-2 available-list">
        <div 
          v-for="ch in availableChannels" 
          :key="ch.ID"
          class="channel-item p-1 hover:bg-gray-200 cursor-pointer"
          :class="{ 'bg-blue-100': selectedAvailable === ch.ID }"
          @click="selectedAvailable = ch.ID"
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
          class="channel-item p-1 hover:bg-gray-200 cursor-pointer flex justify-between"
          :class="{ 'bg-blue-100': selectedMemberIndex === index }"
          @click="selectedMemberIndex = index"
        >
          <span>{{ index + 1 }}. {{ ch.Name }}</span>
        </div>
      </div>
    </div>

    <!-- Reorder Controls -->
    <div class="flex flex-col justify-center gap-2">
      <button class="btn-up px-3 py-1 bg-gray-200 rounded hover:bg-gray-300" @click="moveUp">Up</button>
      <button class="btn-down px-3 py-1 bg-gray-200 rounded hover:bg-gray-300" @click="moveDown">Down</button>
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

const selectedAvailable = ref(null)
const selectedMemberIndex = ref(null)

const availableChannels = computed(() => {
  const memberIDs = new Set(localZone.value.Channels.map(c => c.ID))
  return props.allChannels.filter(c => !memberIDs.has(c.ID))
})

function addToZone() {
  if (selectedAvailable.value) {
    const ch = props.allChannels.find(c => c.ID === selectedAvailable.value)
    if (ch) {
      localZone.value.Channels.push(ch)
      selectedAvailable.value = null
      emitUpdate()
    }
  }
}

function removeFromZone() {
  if (selectedMemberIndex.value !== null) {
    localZone.value.Channels.splice(selectedMemberIndex.value, 1)
    selectedMemberIndex.value = null
    emitUpdate()
  }
}

function moveUp() {
  const idx = selectedMemberIndex.value
  if (idx !== null && idx > 0) {
    const item = localZone.value.Channels[idx]
    localZone.value.Channels.splice(idx, 1)
    localZone.value.Channels.splice(idx - 1, 0, item)
    selectedMemberIndex.value = idx - 1
    emitUpdate()
  }
}

function moveDown() {
  const idx = selectedMemberIndex.value
  if (idx !== null && idx < localZone.value.Channels.length - 1) {
    const item = localZone.value.Channels[idx]
    localZone.value.Channels.splice(idx, 1)
    localZone.value.Channels.splice(idx + 1, 0, item)
    selectedMemberIndex.value = idx + 1
    emitUpdate()
  }
}

function emitUpdate() {
  emit('update:modelValue', localZone.value)
}
</script>
