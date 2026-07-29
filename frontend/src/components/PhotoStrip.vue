<script setup lang="ts">
import type { PhotoSubmission } from '@/types'

withDefaults(
  defineProps<{ photos: PhotoSubmission[]; total: number; title?: string; showSlots?: boolean }>(),
  { title: '最快交卷的前 5 張', showSlots: true },
)
defineEmits<{ (e: 'open', index: number): void }>()

const medals = ['🥇', '🥈', '🥉', '4️⃣', '5️⃣']
</script>

<template>
  <section class="pointer-events-auto">
    <header class="mb-2 flex items-center gap-3 px-1">
      <h2 class="text-sm font-bold text-slate-300">{{ title }}</h2>
      <span class="text-xs text-slate-500">共 {{ total }} 張投稿 · 點一下看大圖</span>
    </header>

    <ul class="flex gap-3 overflow-x-auto pb-1">
      <li v-for="(photo, index) in photos" :key="photo.photoId">
        <button
          class="group relative block w-28 overflow-hidden rounded-2xl border-2 border-white/20 bg-slate-900 transition hover:border-blush-400 lg:w-36"
          @click="$emit('open', index)"
        >
          <img
            :src="photo.url"
            :alt="`${photo.playerName} 的投稿`"
            class="aspect-square w-full object-cover"
          />
          <span class="absolute left-1.5 top-1.5 text-xl drop-shadow">{{ medals[index] }}</span>
          <span
            class="block truncate bg-black/70 px-2 py-1 text-[11px] font-medium text-white"
          >
            {{ photo.playerAvatar }} {{ photo.playerName }}
          </span>
        </button>
      </li>

      <li
        v-for="n in showSlots ? Math.max(0, 5 - photos.length) : 0"
        :key="`empty-${n}`"
        class="grid w-28 place-items-center rounded-2xl border-2 border-dashed border-white/10 text-white/20 lg:w-36"
      >
        <span class="aspect-square w-full place-content-center text-center text-3xl leading-[7rem] lg:leading-[9rem]">
          ·
        </span>
      </li>
    </ul>
  </section>
</template>
