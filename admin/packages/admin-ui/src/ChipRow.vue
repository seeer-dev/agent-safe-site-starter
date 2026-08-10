<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <div class="chip-row" :class="$props.class"><slot /></div>
</template>
