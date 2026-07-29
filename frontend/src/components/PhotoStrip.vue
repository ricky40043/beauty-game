<script setup lang="ts">
import { computed } from 'vue'
import type { PhotoSubmission } from '@/types'

const props = withDefaults(
  defineProps<{
    photos: PhotoSubmission[]
    total: number
    /** 這一排總共留幾格。用格數而不是固定寬度，張數多時縮小而不是變成要橫向捲動 */
    limit?: number
    title?: string
    showSlots?: boolean
  }>(),
  { limit: 5, title: '', showSlots: true },
)

defineEmits<{ (e: 'open', index: number): void }>()

const heading = computed(() => props.title || `最快交卷的前 ${props.limit} 張`)

const emptySlots = computed(() =>
  props.showSlots ? Math.max(0, props.limit - props.photos.length) : 0,
)

// 前三名用獎牌，之後直接標名次，才不會被表情符號數量限制住
const badge = (index: number) => ['🥇', '🥈', '🥉'][index] ?? `#${index + 1}`

// 張數越多字要越小，不然縮圖被文字撐開
const dense = computed(() => props.limit > 10)
</script>

<template>
  <section class="pointer-events-auto">
    <header class="mb-2 flex items-center gap-3 px-1">
      <h2 class="text-sm font-bold text-slate-300">{{ heading }}</h2>
      <span class="text-xs text-slate-500">共 {{ total }} 張投稿 · 點一下看大圖</span>
    </header>

    <!--
      用 grid 讓所有格子平均分掉整個寬度：張數拉到 20 時縮圖會自己變小，
      而不是溢出成需要橫向捲動 —— 主畫面通常投在電視上，沒辦法捲。
    -->
    <ul
      class="grid gap-2"
      :style="{ gridTemplateColumns: `repeat(${Math.max(limit, photos.length)}, minmax(0, 1fr))` }"
    >
      <li v-for="(photo, index) in photos" :key="photo.photoId">
        <button
          class="relative block w-full overflow-hidden rounded-xl border-2 border-white/20 bg-slate-900 transition hover:border-blush-400"
          @click="$emit('open', index)"
        >
          <img
            :src="photo.url"
            :alt="`${photo.playerName} 的投稿`"
            class="aspect-square w-full object-cover"
          />
          <span
            class="absolute left-1 top-1 rounded bg-black/60 px-1 font-bold drop-shadow"
            :class="dense ? 'text-xs' : 'text-lg'"
          >
            {{ badge(index) }}
          </span>
          <span
            class="block truncate bg-black/70 px-1.5 py-0.5 font-medium text-white"
            :class="dense ? 'text-[9px]' : 'text-[11px]'"
          >
            {{ photo.playerAvatar }} {{ photo.playerName }}
          </span>
        </button>
      </li>

      <li
        v-for="n in emptySlots"
        :key="`empty-${n}`"
        class="grid aspect-square place-items-center rounded-xl border-2 border-dashed border-white/10 text-white/15"
      >
        ·
      </li>
    </ul>
  </section>
</template>
