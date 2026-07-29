<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { PhotoSubmission } from '@/types'

const props = defineProps<{ photos: PhotoSubmission[]; questionText: string }>()
const emit = defineEmits<{
  (e: 'confirm', photoIds: string[]): void
  (e: 'skip'): void
  (e: 'preview', index: number): void
}>()

const MAX = 5

const picked = ref<string[]>([])

// 換題目時把上一題的選擇清乾淨
watch(
  () => props.questionText,
  () => (picked.value = []),
)

const rankOf = (photoId: string) => {
  const index = picked.value.indexOf(photoId)
  return index >= 0 ? index + 1 : 0
}

const toggle = (photoId: string) => {
  const index = picked.value.indexOf(photoId)
  if (index >= 0) {
    picked.value.splice(index, 1)
    return
  }
  if (picked.value.length >= MAX) return
  picked.value.push(photoId)
}

const medal = (rank: number) => ['🥇', '🥈', '🥉', '4️⃣', '5️⃣'][rank - 1] ?? ''
const points = (rank: number) => [100, 80, 60, 40, 20][rank - 1] ?? 0

const canConfirm = computed(() => picked.value.length > 0)
</script>

<template>
  <section class="flex h-full flex-col">
    <header class="flex flex-wrap items-baseline justify-between gap-3">
      <div>
        <h2 class="text-2xl font-black">選出今日最美</h2>
        <p class="mt-1 text-sm text-slate-400">
          點選照片決定名次（最多 5 張），點選的順序就是名次。
        </p>
      </div>
      <p class="text-sm text-slate-400">
        已選 <span class="font-bold text-blush-300">{{ picked.length }}</span> / {{ MAX }}
      </p>
    </header>

    <ul
      class="mt-4 grid min-h-0 flex-1 grid-cols-2 gap-3 overflow-y-auto pr-1 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5"
    >
      <li v-for="(photo, index) in photos" :key="photo.photoId" class="relative">
        <button
          class="block w-full overflow-hidden rounded-2xl border-2 transition"
          :class="
            rankOf(photo.photoId)
              ? 'border-blush-400 ring-2 ring-blush-400/50'
              : 'border-white/15 hover:border-white/40'
          "
          @click="toggle(photo.photoId)"
        >
          <img
            :src="photo.url"
            :alt="`${photo.playerName} 的投稿`"
            class="aspect-square w-full object-cover"
          />
          <span class="flex items-center justify-between gap-1 bg-black/70 px-2 py-1.5 text-xs">
            <span class="truncate">{{ photo.playerAvatar }} {{ photo.playerName }}</span>
            <span class="shrink-0 text-slate-400">#{{ photo.order }}</span>
          </span>
        </button>

        <span
          v-if="rankOf(photo.photoId)"
          class="pointer-events-none absolute left-2 top-2 rounded-xl bg-blush-500 px-2 py-1 text-sm font-bold shadow-lg"
        >
          {{ medal(rankOf(photo.photoId)) }} +{{ points(rankOf(photo.photoId)) }}
        </span>

        <button
          class="absolute right-2 top-2 rounded-lg bg-black/60 px-2 py-1 text-xs backdrop-blur"
          aria-label="放大這張"
          @click.stop="emit('preview', index)"
        >
          🔍
        </button>
      </li>
    </ul>

    <footer class="mt-4 flex shrink-0 gap-3">
      <button class="btn-ghost px-6 py-3" @click="emit('skip')">這題都不選</button>
      <button
        class="btn-primary flex-1 py-3 text-lg"
        :disabled="!canConfirm"
        @click="emit('confirm', [...picked])"
      >
        公布結果（{{ picked.length }} 位得獎）
      </button>
    </footer>
  </section>
</template>
