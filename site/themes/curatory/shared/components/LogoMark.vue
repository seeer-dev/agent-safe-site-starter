<script setup lang="ts">
const brandAsset = '/assets/images/' + 'curatory-bloom-mark.svg'

withDefaults(defineProps<{ size?: number; light?: boolean }>(), {
  size: 40,
  light: false,
})
</script>

<template>
  <img
    :src="brandAsset"
    :width="size"
    :height="size"
    :class="{ 'logo-mark-light': light }"
    class="shrink-0 object-contain"
    alt="質選所花間標記"
  />
</template>

<style scoped>
.logo-mark-light {
  filter: brightness(0) saturate(0) invert(94%) sepia(13%) saturate(389%) hue-rotate(345deg) brightness(102%) contrast(91%);
}
</style>
