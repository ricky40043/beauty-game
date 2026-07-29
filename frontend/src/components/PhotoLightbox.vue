<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import type { PhotoSubmission } from '@/types'

const props = defineProps<{ photos: PhotoSubmission[]; index: number }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'update:index', value: number): void }>()

const current = computed(() => props.photos[props.index])

const step = (delta: number) => {
  if (!props.photos.length) return
  const next = (props.index + delta + props.photos.length) % props.photos.length
  emit('update:index', next)
}

const onKey = (event: KeyboardEvent) => {
  if (event.key === 'Escape') emit('close')
  if (event.key === 'ArrowRight') step(1)
  if (event.key === 'ArrowLeft') step(-1)
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <div
    v-if="current"
    class="fixed inset-0 z-[60] flex flex-col items-center justify-center gap-4 bg-slate-950/95 p-6 backdrop-blur"
    @click.self="$emit('close')"
  >
    <img
      :src="current.url"
      :alt="`${current.playerName} 的投稿`"
      class="max-h-[72vh] max-w-full rounded-3xl border-4 border-white/90 object-contain shadow-2xl"
    />

    <div class="text-center">
      <p class="text-xl font-bold">{{ current.playerAvatar }} {{ current.playerName }}</p>
      <p class="mt-1 text-sm text-slate-400">
        第 {{ current.order }} 個交卷 · {{ current.elapsedSec.toFixed(1) }} 秒
      </p>
    </div>

    <div class="flex items-center gap-3">
      <button class="btn-ghost px-4 py-2" @click="step(-1)">← 上一張</button>
      <span class="text-sm text-slate-400">{{ index + 1 }} / {{ photos.length }}</span>
      <button class="btn-ghost px-4 py-2" @click="step(1)">下一張 →</button>
    </div>

    <button class="btn-ghost px-6 py-2" @click="$emit('close')">關閉</button>
  </div>
</template>
