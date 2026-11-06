<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import PhotoLightbox from '@/components/PhotoLightbox.vue'
import type { PhotoSubmission } from '@/types'

const props = defineProps<{
  photos: PhotoSubmission[]
  questionText: string
  /** 0 = 不限，全部投稿都能排名 */
  maxWinners: number
  /** 房主拍照中已經投好的名次（photoId 依名次排），進評選時直接帶入 */
  initialPicked?: string[]
}>()
const emit = defineEmits<{
  (e: 'confirm', photoIds: string[]): void
  (e: 'skip'): void
}>()

const limit = computed(() => (props.maxWinners > 0 ? props.maxWinners : props.photos.length))

const picked = ref<string[]>([...(props.initialPicked ?? [])])

/** 放大中的那張；-1 = 沒有放大 */
const previewIndex = ref(-1)

// 換題目時把上一題的選擇清乾淨
watch(
  () => props.questionText,
  () => {
    picked.value = []
    previewIndex.value = -1
  },
)

const rankOf = (photoId: string) => {
  const index = picked.value.indexOf(photoId)
  return index >= 0 ? index + 1 : 0
}

/**
 * 放大圖下方的大按鈕：投給這張就照點選順序拿下一個名次（第一個投的就是第 1 名），
 * 已經選過的再按一次是取消。投完直接回到列表，看得到名次徽章。
 */
const vote = (photoId: string) => {
  toggle(photoId)
  previewIndex.value = -1
}

const toggle = (photoId: string) => {
  const index = picked.value.indexOf(photoId)
  if (index >= 0) {
    picked.value.splice(index, 1)
    return
  }
  if (picked.value.length >= limit.value) return
  picked.value.push(photoId)
}

// 跟後端 services.PointsForRank 對齊：前 5 名照表，第 6 名起 10 分
const points = (rank: number) => [100, 80, 60, 40, 20][rank - 1] ?? 10

const canConfirm = computed(() => picked.value.length > 0)
</script>

<template>
  <section class="flex h-full flex-col">
    <header class="flex flex-wrap items-baseline justify-between gap-3">
      <div>
        <h2 class="text-2xl font-black">選出今日最美</h2>
        <p class="mt-1 text-sm text-slate-400">
          點照片放大，按「投票給這張」決定名次（{{ maxWinners > 0 ? `最多 ${maxWinners} 張` : '全部都可以排' }}），投的順序就是名次。
        </p>
      </div>
      <p class="text-sm text-slate-400">
        已選 <span class="font-bold text-blush-300">{{ picked.length }}</span> / {{ limit }}
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
          @click="previewIndex = index"
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

        <!-- 名次要一眼看得出來：大數字為主，分數只是小字附註 -->
        <span
          v-if="rankOf(photo.photoId)"
          class="pointer-events-none absolute left-2 top-2 flex items-center gap-1.5 rounded-2xl bg-blush-500 py-1 pl-1 pr-3 shadow-lg"
        >
          <span
            class="grid h-11 min-w-[2.75rem] place-items-center rounded-xl bg-white px-1.5 text-3xl font-black tabular-nums text-blush-600"
          >
            {{ rankOf(photo.photoId) }}
          </span>
          <span class="text-left leading-tight">
            <span class="block text-sm font-bold">第 {{ rankOf(photo.photoId) }} 名</span>
            <span class="block text-[11px] opacity-80">+{{ points(rankOf(photo.photoId)) }} 分</span>
          </span>
        </span>
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
        <button
          v-if="rankOf(photo.photoId)"
          class="btn-ghost w-full max-w-md py-5 text-2xl"
          @click="vote(photo.photoId)"
        >
          取消選擇（目前第 {{ rankOf(photo.photoId) }} 名）
        </button>
        <button
          v-else
          class="btn-primary w-full max-w-md py-5 text-2xl"
          :disabled="picked.length >= limit"
          @click="vote(photo.photoId)"
        >
          {{ picked.length >= limit ? `名額已滿（最多 ${limit} 名）` : `🗳 投票給這張（第 ${picked.length + 1} 名）` }}
        </button>
      </template>
    </PhotoLightbox>

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
