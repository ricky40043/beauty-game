<script setup lang="ts">
import { ref } from 'vue'
import PhotoLightbox from '@/components/PhotoLightbox.vue'
import type { PhotoSubmission } from '@/types'

/** 手機上的投票清單：每人一票、可改投、不能投自己 */
defineProps<{
  photos: PhotoSubmission[]
  counts: Record<string, number>
  myVote: string
  myId: string
}>()

const emit = defineEmits<{ vote: [targetPlayerId: string] }>()

/** 點照片全螢幕放大，下面就是大大的投票鈕 */
const previewIndex = ref(-1)

const voteAndClose = (target: string) => {
  emit('vote', target)
  previewIndex.value = -1
}
</script>

<template>
  <div v-if="!photos.length" class="grid flex-1 place-items-center text-center text-sm text-slate-400">
    還沒有人上傳，等第一張照片出現就能投票
  </div>
  <ul v-else class="grid grid-cols-2 gap-3">
    <li
      v-for="(photo, index) in photos"
      :key="photo.playerId"
      class="overflow-hidden rounded-2xl border-2 bg-slate-900"
      :class="myVote === photo.playerId ? 'border-blush-400 ring-2 ring-blush-400/50' : 'border-white/10'"
    >
      <button class="block w-full" @click="previewIndex = index">
        <img :src="photo.url" :alt="`${photo.playerName} 的投稿`" class="aspect-square w-full object-cover" />
      </button>
      <div class="flex items-center justify-between gap-1 px-2 pt-1.5 text-xs">
        <span class="truncate">{{ photo.playerAvatar }} {{ photo.playerName }}</span>
        <span class="shrink-0 font-bold text-blush-300">🗳 {{ counts[photo.playerId] ?? 0 }}</span>
      </div>
      <div class="p-2">
        <button v-if="photo.playerId === myId" class="btn-ghost w-full py-2 text-xs" disabled>你自己</button>
        <button
          v-else-if="myVote === photo.playerId"
          class="btn-primary w-full py-2 text-sm"
          @click="emit('vote', '')"
        >
          ✅ 已投（點一下收回）
        </button>
        <button v-else class="btn-ghost w-full py-2 text-sm" @click="emit('vote', photo.playerId)">
          投這張
        </button>
      </div>
    </li>
  </ul>

  <PhotoLightbox
    v-if="previewIndex >= 0"
    :photos="photos"
    :index="previewIndex"
    @update:index="previewIndex = $event"
    @close="previewIndex = -1"
  >
    <template #default="{ photo }">
      <button v-if="photo.playerId === myId" class="btn-ghost w-full max-w-md py-5 text-xl" disabled>
        這是你自己的照片
      </button>
      <button
        v-else-if="myVote === photo.playerId"
        class="btn-ghost w-full max-w-md py-5 text-xl"
        @click="voteAndClose('')"
      >
        ✅ 已投給這張（點一下取消）
      </button>
      <button v-else class="btn-primary w-full max-w-md py-5 text-2xl" @click="voteAndClose(photo.playerId)">
        🗳 投票給這個人
      </button>
    </template>
  </PhotoLightbox>
</template>
