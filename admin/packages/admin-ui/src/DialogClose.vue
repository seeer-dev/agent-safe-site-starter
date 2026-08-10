<script setup lang="ts">
import { inject } from 'vue'

const setOpen = inject<(v: boolean) => void>('dialog-set-open')!
</script>

<template>
  <span
    data-slot="dialog-close"
    style="display: contents"
    @click="setOpen(false)"
  >
    <slot />
  </span>
</template>
