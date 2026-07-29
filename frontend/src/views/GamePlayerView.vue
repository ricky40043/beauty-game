<script setup lang="ts">
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import CameraCapture from '@/components/CameraCapture.vue'
import { uploadPhoto } from '@/services/api'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'
import { CATEGORY_LABELS } from '@/types'

const game = useGameStore()
const socket = useSocketStore()
const ui = useUIStore()
const router = useRouter()

const {
  roomId,
  mode,
  status,
  question,
  timeLeft,
  myPhotoUrl,
  uploading,
  roundResult,
  scores,
  playerId,
} = storeToRefs(game)

const camera = ref<InstanceType<typeof CameraCapture> | null>(null)

const canShoot = computed(() => status.value === 'shooting')

const urgency = computed(() => {
  if (!question.value) return 'text-slate-300'
  if (timeLeft.value <= 5) return 'text-rose-400'
  if (timeLeft.value <= 15) return 'text-amber-300'
  return 'text-emerald-300'
})

const myScore = computed(() => scores.value.find((s) => s.playerId === playerId.value))

const myWin = computed(() =>
  roundResult.value?.winners.find((w) => w.playerId === playerId.value),
)

const submit = async (blob: Blob) => {
  if (!canShoot.value) {
    ui.showError('這題已經收桌了')
    return
  }

  uploading.value = true
  try {
    const photoId = await uploadPhoto(roomId.value, blob)
    socket.submitPhoto(photoId)
  } catch (error) {
    uploading.value = false
    ui.showError(error instanceof Error ? error.message : '上傳失敗')
  }
}
</script>

<template>
  <main class="flex min-h-screen flex-col px-4 pb-5 pt-4">
    <!-- 題目永遠釘在最上方 -->
    <header class="shrink-0 rounded-3xl border border-white/10 bg-slate-900/80 p-4 backdrop-blur">
      <div v-if="question" class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <p class="text-xs text-slate-400">
            第 {{ question.questionNum }} / {{ question.totalQuestions }} 題
            <span class="ml-1 rounded bg-white/10 px-1.5 py-0.5">
              {{ CATEGORY_LABELS[question.category] ?? question.category }}
            </span>
            <span v-if="mode === 'group'" class="ml-1 rounded bg-blush-500/25 px-1.5 py-0.5">
              合體照
            </span>
          </p>
          <h1 class="mt-1.5 text-xl font-black leading-snug">{{ question.text }}</h1>
        </div>
        <div class="shrink-0 text-right">
          <p class="text-3xl font-black tabular-nums" :class="urgency">{{ timeLeft }}</p>
          <p class="text-[10px] text-slate-500">秒</p>
        </div>
      </div>

      <div v-else class="text-center text-sm text-slate-400">等待下一題…</div>
    </header>

    <!-- 拍照階段 -->
    <section v-if="canShoot" class="mt-4 flex min-h-0 flex-1 flex-col">
      <div
        v-if="myPhotoUrl"
        class="mb-3 flex items-center gap-3 rounded-2xl bg-emerald-500/15 px-4 py-3 text-sm"
      >
        <img :src="myPhotoUrl" alt="你上傳的照片" class="h-12 w-12 rounded-xl object-cover" />
        <span class="flex-1 text-emerald-100">
          已上傳！{{ mode === 'group' ? '還能再拍一張不同的' : '不滿意可以重拍覆蓋' }}
        </span>
      </div>

      <CameraCapture ref="camera" class="min-h-0 flex-1" :busy="uploading" @submit="submit" />
    </section>

    <!-- 評選中 -->
    <section v-else-if="status === 'judging'" class="mt-6 flex flex-1 flex-col items-center justify-center gap-3 text-center">
      <div class="text-6xl">🧐</div>
      <h2 class="text-xl font-bold">主持人評選中…</h2>
      <p class="text-sm text-slate-400">抬頭看主畫面！</p>
      <img
        v-if="myPhotoUrl"
        :src="myPhotoUrl"
        alt="你的投稿"
        class="mt-2 w-40 rounded-2xl object-cover shadow-xl"
      />
    </section>

    <!-- 本題結果 -->
    <section v-else-if="status === 'round_result'" class="mt-6 flex flex-1 flex-col items-center justify-center gap-3 text-center">
      <template v-if="myWin">
        <div class="text-6xl">🏆</div>
        <h2 class="text-2xl font-black text-blush-300">第 {{ myWin.rank }} 名！</h2>
        <p class="text-lg font-bold text-emerald-300">+{{ myWin.points }} 分</p>
      </template>
      <template v-else>
        <div class="text-6xl">🙈</div>
        <h2 class="text-xl font-bold">這題沒被選上</h2>
        <p class="text-sm text-slate-400">下一題再拚一次！</p>
      </template>

      <p v-if="roundResult?.groupBonus" class="text-sm text-sky-300">
        團體合作分 +{{ roundResult.groupBonus }}（全場都有）
      </p>

      <div v-if="myScore" class="card mt-4 w-full max-w-xs">
        <p class="text-sm text-slate-400">目前總分</p>
        <p class="text-3xl font-black">{{ myScore.score }}</p>
        <p class="mt-1 text-sm text-slate-400">第 {{ myScore.rank }} 名</p>
      </div>
    </section>

    <!-- 這場已結束（例如按上一頁回到這裡），別讓畫面卡在空白 -->
    <section
      v-else-if="status === 'finished'"
      class="mt-6 flex flex-1 flex-col items-center justify-center gap-4 text-center"
    >
      <div class="text-6xl">🏁</div>
      <h2 class="text-xl font-bold">這一場結束了</h2>
      <p class="text-sm text-slate-400">房間還在，等主持人開下一局。</p>
      <button class="btn-primary mt-2 px-8 py-3" @click="router.push(`/results/${roomId}`)">
        看結算
      </button>
    </section>

    <section v-else class="mt-6 flex flex-1 items-center justify-center text-sm text-slate-400">
      等主持人操作…
    </section>
  </main>
</template>
