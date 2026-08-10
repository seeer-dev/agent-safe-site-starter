<script setup lang="ts">
import { inject } from 'vue'

const setOpen = inject<(v: boolean) => void>('sheet-set-open')!
</script>

<template>
  <span
    data-slot="sheet-trigger"
    style="display: contents"
    @click="setOpen(true)"
  >
    <slot />
  </span>
</template>
