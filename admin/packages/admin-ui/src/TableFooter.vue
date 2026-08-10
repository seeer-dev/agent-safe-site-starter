<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <footer class="tfoot" :class="$props.class"><slot /></footer>
</template>
