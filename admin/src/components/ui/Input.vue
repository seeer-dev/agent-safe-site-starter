<script setup lang="ts">
import { ref, computed } from 'vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  type?: string
  placeholder?: string
  disabled?: boolean
  width?: string
  id?: string
  name?: string
  autocomplete?: string
  required?: boolean
  toggle?: boolean
}>(), {
  modelValue: '',
  type: 'text',
  placeholder: '',
  disabled: false,
  width: '100%',
  id: '',
  name: '',
  autocomplete: '',
  required: false,
  toggle: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const showPassword = ref(false)
const effectiveType = computed(() => {
  if (props.toggle && props.type === 'password') {
    return showPassword.value ? 'text' : 'password'
  }
  return props.type
})

function onInput(ev: Event) {
  emit('update:modelValue', (ev.target as HTMLInputElement).value)
}
</script>

<template>
  <div v-if="toggle && type === 'password'" class="input-toggle-wrap">
    <input
      :id="id || undefined"
      :name="name || undefined"
      :type="effectiveType"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :autocomplete="autocomplete || undefined"
      :required="required || undefined"
      class="inp"
      :style="{ width }"
      @input="onInput"
    />
    <button
      type="button"
      tabindex="-1"
      class="eye-toggle"
      :aria-label="showPassword ? '隱藏密碼' : '顯示密碼'"
      @click="showPassword = !showPassword"
    >
      <svg v-if="showPassword" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/><path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"/><path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"/><line x1="2" x2="22" y1="2" y2="22"/></svg>
      <svg v-else viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>
    </button>
  </div>
  <input
    v-else
    :id="id || undefined"
    :name="name || undefined"
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :autocomplete="autocomplete || undefined"
    :required="required || undefined"
    class="inp"
    :style="{ width }"
    @input="onInput"
  />
</template>
