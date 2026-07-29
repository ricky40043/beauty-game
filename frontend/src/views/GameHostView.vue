<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import JudgePanel from '@/components/JudgePanel.vue'
import PhotoLightbox from '@/components/PhotoLightbox.vue'
import PhotoPopupLayer from '@/components/PhotoPopupLayer.vue'
import PhotoStrip from '@/components/PhotoStrip.vue'
import QRCodeDisplay from '@/components/QRCodeDisplay.vue'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'
import { celebrate } from '@/utils/confetti'
import { on } from '@/utils/bus'
import { CATEGORY_LABELS } from '@/types'

const game = useGameStore()
const socket = useSocketStore()
const router = useRouter()

const {
  roomId,
  joinUrl,
  mode,
  status,
  question,
  timeLeft,
  roundPhotos,
  topFive,
  submittedCount,
  connectedCount,
  roundResult,
  scores,
} = storeToRefs(game)

const lightboxIndex = ref(-1)

// 這個遊戲隨時可以中途加入，所以 QR code 全程都留在右上角
const shareUrl = computed(() => joinUrl.value || `${window.location.origin}/join/${roomId.value}`)
const qrExpanded = ref(false)

// 示範圖只有部分題目有（目前是團體題）。與其先問後端有沒有，
// 直接載入、404 就把區塊收起來，房主之後在後台補圖也會自動出現。
const exampleFailed = ref(false)
const exampleUrl = computed(() =>
  question.value ? `/api/questions/${question.value.questionId}/example` : '',
)
const showExample = computed(() => Boolean(exampleUrl.value) && !exampleFailed.value)

watch(
  () => question.value?.questionId,
  () => (exampleFailed.value = false),
)

const openLightbox = (index: number) => (lightboxIndex.value = index)
const closeLightbox = () => (lightboxIndex.value = -1)

const progress = computed(() => {
  if (!question.value?.timeLimit) return 0
  return Math.max(0, Math.min(1, timeLeft.value / question.value.timeLimit))
})

const urgent = computed(() => timeLeft.value <= 10)

const offRoundResult = on('roundResult', (result) => {
  if (result.winners.length) celebrate()
})

onBeforeUnmount(offRoundResult)

const isLastQuestion = computed(
  () => !!question.value && question.value.questionNum >= question.value.totalQuestions,
)
</script>

<template>
  <main class="flex min-h-screen flex-col px-8 py-6 lg:px-12">
    <!-- 題目列 -->
    <header class="flex shrink-0 flex-wrap items-center justify-between gap-6">
      <div class="min-w-0 flex-1">
        <p class="text-sm text-slate-400">
          第 {{ question?.questionNum ?? '-' }} / {{ question?.totalQuestions ?? '-' }} 題
          <span v-if="question" class="ml-2 rounded bg-white/10 px-2 py-0.5">
            {{ CATEGORY_LABELS[question.category] ?? question.category }}
          </span>
          <span v-if="mode === 'group'" class="ml-2 rounded bg-blush-500/25 px-2 py-0.5">
            團體合體照
          </span>
        </p>
        <h1 class="mt-2 text-4xl font-black leading-tight lg:text-6xl">
          {{ question?.text ?? '準備中…' }}
        </h1>
      </div>

      <div class="flex shrink-0 items-center gap-6">
        <div v-if="status === 'shooting'" class="text-center">
          <p class="text-sm text-slate-400">已交卷</p>
          <p class="text-3xl font-black tabular-nums">
            {{ submittedCount }}<span class="text-lg text-slate-500">/{{ connectedCount }}</span>
          </p>
        </div>

        <div v-if="status === 'shooting'" class="relative grid h-28 w-28 place-items-center">
          <svg class="absolute inset-0 -rotate-90" viewBox="0 0 100 100">
            <circle cx="50" cy="50" r="44" fill="none" stroke="rgba(255,255,255,0.1)" stroke-width="8" />
            <circle
              cx="50"
              cy="50"
              r="44"
              fill="none"
              :stroke="urgent ? '#fb7185' : '#f92e83'"
              stroke-width="8"
              stroke-linecap="round"
              :stroke-dasharray="2 * Math.PI * 44"
              :stroke-dashoffset="2 * Math.PI * 44 * (1 - progress)"
              class="transition-all duration-1000 ease-linear"
            />
          </svg>
          <span class="text-4xl font-black tabular-nums" :class="urgent ? 'text-rose-400' : ''">
            {{ timeLeft }}
          </span>
        </div>

        <!-- 隨時可中途加入，所以 QR 全程掛在右上角；點一下可放大給遠處的人掃 -->
        <button class="shrink-0 text-center" @click="qrExpanded = true">
          <QRCodeDisplay :value="shareUrl" :size="180" class="w-20 !p-1.5 lg:w-24" />
          <p class="mt-1 text-sm font-bold tracking-widest text-blush-300">{{ roomId }}</p>
          <p class="text-[10px] text-slate-500">掃碼加入</p>
        </button>
      </div>
    </header>

    <!-- 拍照中 -->
    <section v-if="status === 'shooting'" class="mt-6 flex min-h-0 flex-1 flex-col justify-end gap-6">
      <div class="grid min-h-0 flex-1 place-items-center text-center">
        <!-- 示範圖放正中央；照片彈出層 z-50 會疊在它上面 -->
        <figure v-if="showExample" class="flex max-h-full flex-col items-center">
          <p class="mb-1 text-xs uppercase tracking-widest text-slate-500">參考站位</p>
          <img
            :src="exampleUrl"
            alt="這題的參考站位示範圖"
            class="max-h-[46vh] w-auto max-w-full opacity-90"
            @error="exampleFailed = true"
          />
        </figure>

        <div v-else-if="!submittedCount">
          <div class="animate-pulse text-7xl">📸</div>
          <p class="mt-4 text-xl text-slate-400">等大家拍照上傳…</p>
        </div>
      </div>

      <PhotoStrip :photos="topFive" :total="roundPhotos.length" @open="openLightbox" />

      <div class="flex justify-end">
        <button class="btn-ghost px-6 py-3" @click="socket.endShooting()">提前結束拍照</button>
      </div>
    </section>

    <!-- 評選中 -->
    <section v-else-if="status === 'judging'" class="mt-6 min-h-0 flex-1">
      <JudgePanel
        :photos="roundPhotos"
        :question-text="question?.text ?? ''"
        @confirm="socket.pickWinners($event)"
        @skip="socket.skipRound()"
        @preview="openLightbox"
      />
    </section>

    <!-- 公布結果 -->
    <section v-else-if="status === 'round_result'" class="mt-6 flex min-h-0 flex-1 flex-col gap-6">
      <div v-if="!roundResult?.winners.length" class="grid flex-1 place-items-center text-center">
        <div>
          <div class="text-7xl">🫥</div>
          <p class="mt-4 text-xl text-slate-400">這題沒有選出得獎者</p>
        </div>
      </div>

      <div v-else class="flex min-h-0 flex-1 items-end justify-center gap-6">
        <figure
          v-for="winner in roundResult.winners"
          :key="winner.photoId"
          class="animate-pop-in overflow-hidden rounded-3xl border-4 border-white/90 bg-slate-900 shadow-2xl"
          :class="winner.rank === 1 ? 'w-72 lg:w-96' : 'w-44 lg:w-60'"
        >
          <img
            :src="winner.url"
            :alt="`${winner.playerName} 的得獎照片`"
            class="aspect-square w-full object-cover"
          />
          <figcaption class="bg-white px-3 py-2 text-center text-slate-900">
            <p class="text-2xl">
              {{ ['🥇', '🥈', '🥉', '4️⃣', '5️⃣'][winner.rank - 1] }}
            </p>
            <p class="truncate font-bold">{{ winner.playerAvatar }} {{ winner.playerName }}</p>
            <p class="text-sm font-semibold text-blush-600">+{{ winner.points }} 分</p>
          </figcaption>
        </figure>
      </div>

      <p v-if="roundResult?.groupBonus" class="text-center text-lg text-sky-300">
        🤝 團體合作分：全場每人 +{{ roundResult.groupBonus }} 分
      </p>

      <div class="flex items-end justify-between gap-6">
        <ol class="flex flex-wrap gap-2">
          <li
            v-for="score in scores.slice(0, 6)"
            :key="score.playerId"
            class="flex items-center gap-2 rounded-2xl bg-white/5 px-3 py-2"
          >
            <span class="text-sm font-bold text-slate-400">{{ score.rank }}</span>
            <span class="text-lg">{{ score.avatar }}</span>
            <span class="font-medium">{{ score.playerName }}</span>
            <span class="font-black tabular-nums text-blush-300">{{ score.score }}</span>
            <span v-if="score.gained" class="text-xs text-emerald-300">+{{ score.gained }}</span>
          </li>
        </ol>

        <button class="btn-primary shrink-0 px-8 py-4 text-lg" @click="socket.nextQuestion()">
          {{ isLastQuestion ? '看最終結果 →' : '下一題 →' }}
        </button>
      </div>
    </section>

    <!-- 這場已經結束（例如從結算頁按上一頁回到這裡），給一條路走回去 -->
    <section
      v-else-if="status === 'finished'"
      class="mt-6 flex flex-1 flex-col items-center justify-center gap-4 text-center"
    >
      <div class="text-7xl">🏁</div>
      <h2 class="text-2xl font-bold">這一場已經結束了</h2>
      <p class="text-sm text-slate-400">房間還在，可以看結算或直接開新的一局。</p>
      <div class="mt-2 flex flex-wrap justify-center gap-3">
        <button class="btn-primary px-8 py-3" @click="router.push(`/results/${roomId}`)">
          看結算
        </button>
        <button class="btn-ghost px-8 py-3" @click="socket.resetRoomToLobby()">開新的一局</button>
      </div>
    </section>

    <!-- 照片一到就彈出，停 3 秒淡出，新的直接蓋上舊的 -->
    <PhotoPopupLayer v-if="status === 'shooting'" />

    <!-- QR 放大：房間太大時給遠一點的人掃 -->
    <div
      v-if="qrExpanded"
      class="fixed inset-0 z-[60] flex flex-col items-center justify-center gap-6 bg-slate-950/95 p-8 backdrop-blur"
      @click="qrExpanded = false"
    >
      <QRCodeDisplay :value="shareUrl" :size="640" class="w-[min(70vh,70vw)]" />
      <div class="text-center">
        <p class="text-6xl font-black tracking-[0.2em] text-blush-300">{{ roomId }}</p>
        <p class="mt-2 text-slate-400">中途也能加入，掃碼就開始玩</p>
      </div>
      <p class="text-sm text-slate-500">點任一處關閉</p>
    </div>

    <PhotoLightbox
      v-if="lightboxIndex >= 0"
      :photos="roundPhotos"
      :index="lightboxIndex"
      @update:index="lightboxIndex = $event"
      @close="closeLightbox"
    />
  </main>
</template>
