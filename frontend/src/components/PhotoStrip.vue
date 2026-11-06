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
    /** 標題旁的操作提示 */
    hint?: string
    /** 投票模式：各照片主人的即時票數 */
    voteCounts?: Record<string, number>
    /** 房主邊拍邊排名次：照片主人 playerId → 名次；有傳才顯示「投給它」鈕 */
    picks?: Record<string, number>
    /** 名次已經排滿，不能再投新的 */
    picksFull?: boolean
  }>(),
  { limit: 5, title: '', showSlots: true, hint: '點一下看大圖' },
)

defineEmits<{ (e: 'open', index: number): void; (e: 'pick', playerId: string): void }>()

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
      <span class="text-xs text-slate-500">共 {{ total }} 張投稿 · {{ hint }}</span>
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
            v-if="voteCounts"
            class="absolute right-1 top-1 rounded-lg bg-blush-500 px-1.5 font-black shadow"
            :class="dense ? 'text-xs' : 'text-lg'"
          >
            🗳 {{ voteCounts[photo.playerId] ?? 0 }}
          </span>
          <span
            class="block truncate bg-black/70 px-1.5 py-0.5 font-medium text-white"
            :class="dense ? 'text-[9px]' : 'text-[11px]'"
          >
            {{ photo.playerAvatar }} {{ photo.playerName }}
          </span>
          <span
            v-if="picks?.[photo.playerId]"
            class="absolute right-1 top-1 grid min-w-[2rem] place-items-center rounded-lg bg-blush-500 px-1.5 font-black shadow"
            :class="dense ? 'text-sm' : 'text-2xl'"
          >
            {{ picks[photo.playerId] }}
          </span>
        </button>
        <button
          v-if="picks"
          class="mt-1 w-full rounded-lg py-1.5 font-bold transition"
          :class="[
            picks[photo.playerId] ? 'bg-blush-500 text-white' : 'bg-white/10 text-slate-200 hover:bg-white/20',
            dense ? 'text-[10px]' : 'text-sm',
          ]"
          :disabled="!picks[photo.playerId] && picksFull"
          @click="$emit('pick', photo.playerId)"
        >
          {{ picks[photo.playerId] ? `第 ${picks[photo.playerId]} 名 ✕` : picksFull ? '名額已滿' : '投給它' }}
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
