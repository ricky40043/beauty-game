<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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
  playerName,
  avatar,
  isPractice,
} = storeToRefs(game)

const camera = ref<InstanceType<typeof CameraCapture> | null>(null)

// 遊戲進行中也要能離開房間。手機版面很擠，所以收在一個選單裡而不是常駐按鈕。
const menuOpen = ref(false)

const leave = () => {
  if (window.confirm('離開後這場的分數就不算了，確定要離開嗎？')) {
    socket.leaveRoom()
  }
}

const canShoot = computed(() => status.value === 'shooting')

// 換題時把停在「已拍好等上傳」的畫面切回相機，不用自己按重拍
watch(
  () => question.value?.questionId,
  () => camera.value?.resetForNewQuestion(),
)

const urgency = computed(() => {
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
  <!--
    用 100dvh 而不是 min-h-screen：手機瀏覽器的網址列會伸縮，dvh 會跟著變，
    min-height 則會讓內容超出一個畫面而在底下留一塊黑的。
    再配 overflow-hidden，確保題目、相機、上傳鈕永遠在同一頁看得完。
  -->
  <main
    class="flex h-[100dvh] flex-col overflow-hidden px-3 pt-3"
    style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))"
  >
    <!-- 題目永遠釘在最上方，高度壓到最小 -->
    <header class="shrink-0 rounded-2xl border border-white/10 bg-slate-900/80 px-3 py-2 backdrop-blur">
      <div class="flex items-start justify-between gap-2">
        <div class="min-w-0 flex-1">
          <p class="flex flex-wrap items-center gap-1 text-[11px] leading-none text-slate-400">
            <template v-if="question">
              <span v-if="isPractice" class="rounded bg-amber-500/25 px-1.5 py-0.5 text-amber-200">
                試玩中 · 不計分
              </span>
              <template v-else>
                <span>第 {{ question.questionNum }} / {{ question.totalQuestions }} 題</span>
                <span class="rounded bg-white/10 px-1.5 py-0.5">
                  {{ CATEGORY_LABELS[question.category] ?? question.category }}
                </span>
              </template>
              <span v-if="mode === 'group'" class="rounded bg-blush-500/25 px-1.5 py-0.5">
                合體照
              </span>
            </template>
            <span v-else>等待下一題…</span>
          </p>
          <h1 v-if="question" class="mt-1 text-lg font-black leading-tight">{{ question.text }}</h1>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <div v-if="question && !isPractice" class="text-right leading-none">
            <p class="text-2xl font-black tabular-nums" :class="urgency">{{ timeLeft }}</p>
            <p class="text-[10px] text-slate-500">秒</p>
          </div>

          <button
            class="rounded-lg px-2 py-2 text-lg leading-none text-slate-500 active:bg-white/10"
            aria-label="選單"
            @click="menuOpen = true"
          >
            ⋯
          </button>
        </div>
      </div>
    </header>

    <!-- 拍照階段：相機吃掉所有剩餘空間 -->
    <section v-if="canShoot" class="mt-2 flex min-h-0 flex-1 flex-col">
      <p
        v-if="myPhotoUrl"
        class="mb-1.5 shrink-0 rounded-xl bg-emerald-500/15 px-3 py-1.5 text-center text-xs text-emerald-100"
      >
        ✅ 已上傳，再拍一張就會蓋掉舊的
      </p>

      <CameraCapture ref="camera" class="min-h-0 flex-1" :busy="uploading" @submit="submit" />
    </section>

    <!-- 評選中 -->
    <section
      v-else-if="status === 'judging'"
      class="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 text-center"
    >
      <div class="text-6xl">🧐</div>
      <h2 class="text-xl font-bold">主持人評選中…</h2>
      <p class="text-sm text-slate-400">抬頭看主畫面！</p>
      <img
        v-if="myPhotoUrl"
        :src="myPhotoUrl"
        alt="你的投稿"
        class="max-h-[40vh] w-36 rounded-2xl object-cover shadow-xl"
      />
    </section>

    <!-- 本題結果 -->
    <section
      v-else-if="status === 'round_result'"
      class="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 overflow-y-auto text-center"
    >
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

      <div v-if="myScore" class="card w-full max-w-xs py-3">
        <p class="text-xs text-slate-400">目前總分</p>
        <p class="text-3xl font-black">{{ myScore.score }}</p>
        <p class="mt-0.5 text-sm text-slate-400">第 {{ myScore.rank }} 名</p>
      </div>
    </section>

    <!-- 這場已結束 -->
    <section
      v-else-if="status === 'finished'"
      class="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 text-center"
    >
      <div class="text-6xl">🏁</div>
      <h2 class="text-xl font-bold">這一場結束了</h2>
      <p class="text-sm text-slate-400">房間還在，等主持人開下一局。</p>
      <button class="btn-primary px-8 py-3" @click="router.push(`/results/${roomId}`)">
        看結算
      </button>
    </section>

    <section v-else class="flex min-h-0 flex-1 items-center justify-center text-sm text-slate-400">
      等主持人操作…
    </section>

    <!-- 選單：看自己的資料，以及遊戲中離開房間 -->
    <div
      v-if="menuOpen"
      class="fixed inset-0 z-[70] flex items-end bg-slate-950/70 backdrop-blur-sm"
      @click.self="menuOpen = false"
    >
      <div
        class="w-full rounded-t-3xl border-t border-white/10 bg-slate-900 p-5"
        style="padding-bottom: max(1.25rem, env(safe-area-inset-bottom))"
      >
        <div class="flex items-center gap-3">
          <span class="text-3xl">{{ avatar }}</span>
          <div class="min-w-0 flex-1">
            <p class="truncate font-bold">{{ playerName }}</p>
            <p class="text-xs text-slate-400">房號 {{ roomId }}</p>
          </div>
          <p v-if="myScore" class="text-right">
            <span class="text-xl font-black">{{ myScore.score }}</span>
            <span class="block text-[10px] text-slate-500">目前分數</span>
          </p>
        </div>

        <button class="btn-ghost mt-4 w-full py-3 text-rose-300" @click="leave">離開房間</button>
        <button class="btn-ghost mt-2 w-full py-3" @click="menuOpen = false">取消</button>
      </div>
    </div>
  </main>
</template>
