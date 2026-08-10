<script setup lang="ts">
import { inject, computed, type Ref } from 'vue'

const props = defineProps<{
  value: string
  class?: string
}>()

const selected = inject<Ref<string>>('tabs-selected')!
const isVisible = computed(() => selected.value === props.value)
</script>

<template>
  <div
    v-if="isVisible"
    class="flex-1 outline-none"
    data-slot="tabs-content"
    role="tabpanel"
    :class="$props.class"
  >
    <slot />
  </div>
</template>
