<script setup lang="ts">
import { CATEGORY_LABELS, SOLO_CATEGORIES, type GameMode } from '@/types'

/** 隨機抽題的設定（題數、分類、難度），建房與大廳共用 */
defineProps<{ mode: GameMode }>()

const total = defineModel<number>('total', { required: true })
const categories = defineModel<string[]>('categories', { required: true })
const difficulty = defineModel<'basic' | 'mixed'>('difficulty', { required: true })

const toggleCategory = (category: string) => {
  categories.value = categories.value.includes(category)
    ? categories.value.filter((c) => c !== category)
    : [...categories.value, category]
}
</script>

<template>
  <div class="space-y-4">
    <label class="block">
      <span class="text-sm text-slate-300">題數：{{ total }} 題</span>
      <input
        :value="total"
        type="range"
        min="1"
        max="30"
        class="mt-2 w-full accent-blush-500"
        @input="total = Number(($event.target as HTMLInputElement).value)"
      />
    </label>

    <div v-if="mode === 'solo'">
      <span class="text-sm text-slate-300">題目分類</span>
      <span class="ml-2 text-xs text-slate-500">
        {{ categories.length ? '勾選的分類會平均出題' : '不勾就是全部分類一起抽' }}
      </span>
      <div class="mt-2 flex flex-wrap gap-2">
        <button
          v-for="category in SOLO_CATEGORIES"
          :key="category"
          class="rounded-full px-3 py-1.5 text-xs font-medium transition"
          :class="
            categories.includes(category)
              ? 'bg-blush-500 text-white'
              : 'bg-white/5 text-slate-400 hover:bg-white/10'
          "
          @click="toggleCategory(category)"
        >
          {{ CATEGORY_LABELS[category] }}
        </button>
      </div>
    </div>

    <div class="flex gap-2">
      <button
        class="flex-1 rounded-xl px-3 py-2 text-xs font-medium transition"
        :class="difficulty === 'basic' ? 'bg-white/15 text-white' : 'bg-white/5 text-slate-400 hover:bg-white/10'"
        @click="difficulty = 'basic'"
      >
        只出基礎題
      </button>
      <button
        class="flex-1 rounded-xl px-3 py-2 text-xs font-medium transition"
        :class="difficulty === 'mixed' ? 'bg-white/15 text-white' : 'bg-white/5 text-slate-400 hover:bg-white/10'"
        @click="difficulty = 'mixed'"
      >
        基礎 + 進階混合
      </button>
    </div>
  </div>
</template>
