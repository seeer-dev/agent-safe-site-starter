<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { useToast } from '@/shared/composables/use-toast'
import { cn } from '@/shared/lib/utils'

const { toasts, dismiss } = useToast()
</script>

<template>
  <Teleport to="body">
    <div class="fixed bottom-0 right-0 z-[100] flex flex-col items-end gap-2 p-4 max-w-[440px]">
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          :class="cn(
            'group pointer-events-auto relative flex items-center gap-3 overflow-hidden rounded-full border py-2.5 pl-2.5 pr-9 shadow-lg transition-all',
            t.variant === 'destructive'
              ? 'border-destructive/50 bg-destructive text-destructive-foreground'
              : 'border border-border/60 bg-background text-foreground',
          )"
        >
          <img
            v-if="t.image"
            :src="t.image"
            alt=""
            class="h-10 w-10 shrink-0 rounded-full object-cover ring-2 ring-border/40"
          />
          <div class="min-w-0 flex-1 grid gap-0.5">
            <div v-if="t.title" class="text-sm font-semibold leading-tight">{{ t.title }}</div>
            <div v-if="t.description" class="truncate text-xs opacity-80">{{ t.description }}</div>
          </div>
          <button
            class="absolute right-2 top-1/2 -translate-y-1/2 rounded-full p-1 text-foreground/50 opacity-0 transition-opacity hover:text-foreground focus:opacity-100 group-hover:opacity-100"
            @click="dismiss(t.id)"
          >
            <X class="h-3.5 w-3.5" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
