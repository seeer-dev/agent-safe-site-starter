<script setup lang="ts">
import { computed } from 'vue'
import { Heart, Plus, Star } from 'lucide-vue-next'
import type { Product } from '@/shared/lib/types'
import { useCartStore } from '@/shared/stores/cart'
import { useUiStore } from '@/shared/stores/ui'
import { useToast } from '@/shared/composables/use-toast'
import { formatNTD, cn } from '@/shared/lib/utils'

const props = withDefaults(defineProps<{
  product: Product
  index?: number
}>(), {
  index: 0,
})

const cart = useCartStore()
const ui = useUiStore()
const { toast } = useToast()

const isFavorite = computed(() => cart.favorites.includes(props.product.id))

const discount = computed(() => {
  if (!props.product.originalPrice) return 0
  return Math.round((1 - props.product.price / props.product.originalPrice) * 100)
})

function handleAdd(e: Event) {
  e.stopPropagation()
  cart.addItem(props.product, 1, {
    size: props.product.sizes?.[0]?.value,
    color: props.product.colors?.[0]?.value,
  })
  toast({
    title: '已加入購物車',
    description: `${props.product.name}${props.product.sizes?.[0] ? ` · ${props.product.sizes[0].label}` : ''} · ${formatNTD(props.product.price)}`,
    image: props.product.image,
  })
}

function handleFavorite(e: Event) {
  e.stopPropagation()
  cart.toggleFavorite(props.product.id)
}
</script>

<template>
  <article
    @click="ui.openProductDetail(product)"
    class="card-enter group relative flex cursor-pointer flex-col overflow-hidden rounded-2xl border border-border/60 bg-card transition-colors hover:border-border hover:shadow-lg"
    :style="{ animationDelay: `${Math.min(index * 0.04, 0.24)}s` }"
  >
    <div class="relative aspect-[4/5] overflow-hidden bg-muted/40">
      <img
        :src="product.image"
        :alt="product.name"
        loading="lazy"
        class="h-full w-full object-cover transition-transform duration-700 ease-out group-hover:scale-[1.04]"
      />

      <span
        v-if="product.tag"
        :class="cn(
          'absolute left-3 top-3 inline-flex h-6 items-center rounded-full px-2.5 text-[11px] font-medium',
          product.tag === '特價' ? 'bg-cta text-cta-foreground' : 'bg-background/90 text-foreground backdrop-blur',
        )"
      >
        {{ product.tag }}{{ discount > 0 && product.tag === '特價' ? ` −${discount}%` : '' }}
      </span>

      <span
        v-if="product.stock > 0 && product.stock <= 5"
        class="absolute right-3 top-12 inline-flex h-6 items-center rounded-full bg-red-500 px-2 text-[10px] font-medium text-white backdrop-blur"
      >
        剩 {{ product.stock }} 件
      </span>

      <button
        @click="handleFavorite"
        :aria-label="isFavorite ? '取消收藏' : '加入收藏'"
        class="absolute right-3 top-3 grid h-8 w-8 place-items-center rounded-full bg-background/90 text-foreground backdrop-blur transition-colors hover:bg-background"
      >
        <Heart :class="cn('h-4 w-4 transition-all', isFavorite && 'fill-cta text-cta')" />
      </button>

      <div class="absolute inset-x-3 bottom-3 translate-y-2 opacity-0 transition-all duration-300 group-hover:translate-y-0 group-hover:opacity-100">
        <button
          @click="handleAdd"
          :disabled="product.stock === 0"
          class="flex w-full items-center justify-center gap-2 rounded-full bg-cta py-2.5 text-sm font-medium text-cta-foreground shadow-sm backdrop-blur transition-colors hover:brightness-95 disabled:cursor-not-allowed disabled:opacity-60"
        >
          <Plus class="h-4 w-4" />
          {{ product.stock === 0 ? '已售完' : '快速加入' }}
        </button>
      </div>
    </div>

    <div class="flex flex-1 flex-col p-4">
      <div class="flex items-start justify-between gap-2">
        <h3 class="text-sm font-medium leading-snug">{{ product.name }}</h3>
        <div class="flex items-center gap-1 text-xs text-muted-foreground">
          <Star class="h-3 w-3 fill-cta text-cta" />
          <span>{{ product.rating }}</span>
        </div>
      </div>
      <p class="mt-1.5 line-clamp-2 text-xs leading-relaxed text-muted-foreground">
        {{ product.description }}
      </p>

      <div class="mt-3 flex items-end justify-between">
        <div class="flex items-baseline gap-2">
          <span class="text-base font-semibold tracking-tight text-cta">
            {{ formatNTD(product.price) }}
          </span>
          <span v-if="product.originalPrice" class="text-xs text-muted-foreground line-through">
            {{ formatNTD(product.originalPrice) }}
          </span>
        </div>
        <span class="text-[11px] text-muted-foreground">
          {{ product.reviews }} 則評價
        </span>
      </div>
    </div>
  </article>
</template>
