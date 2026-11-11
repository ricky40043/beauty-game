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
import { useUIStore } from '@/stores/ui'
import { useShutterSound } from '@/composables/useShutterSound'
import { celebrate } from '@/utils/confetti'
import { on } from '@/utils/bus'
import { CATEGORY_EMOJI, CATEGORY_LABELS } from '@/types'

const game = useGameStore()
const ui = useUIStore()
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
  stripLimit,
  stripPhotos,
  submittedCount,
  connectedCount,
  roundResult,
  scores,
  isPractice,
  maxWinners,
  resultSeconds,
  voteMode,
  voteCandidates,
  voteSeconds,
  voteCounts,
} = storeToRefs(game)

/** 這一題走投票模式（試玩不投） */
const voting = computed(() => voteMode.value && !isPractice.value)

/** 投票中依票數排好的候選，同票先交的在前 */
const ranked = computed(() =>
  [...roundPhotos.value].sort(
    (a, b) => (voteCounts.value[b.playerId] ?? 0) - (voteCounts.value[a.playerId] ?? 0) || a.order - b.order,
  ),
)
const totalVotes = computed(() => Object.values(voteCounts.value).reduce((n, v) => n + v, 0))

const lightboxIndex = ref(-1)

// 這個遊戲隨時可以中途加入，所以 QR code 全程都留在右上角
const shareUrl = computed(() => joinUrl.value || `${window.location.origin}/join/${roomId.value}`)
const qrExpanded = ref(false)
const copied = ref(false)

const copyShareUrl = async () => {
  try {
    await navigator.clipboard.writeText(shareUrl.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    // 非 HTTPS 或權限被擋時拿不到剪貼簿，連結文字本身可以手動全選複製
    ui.showError('沒辦法自動複製，請直接選取連結文字')
  }
}

// 「最快交卷」要顯示幾張，1~20 都可以，預設 5
const stripOptions = [1, 3, 5, 8, 10, 12, 15, 20]

// 照片一進來就「喀」一聲。上一聲還在播就停掉重播，連拍時每張都聽得到。
const {
  enabled: soundOn,
  blocked: soundBlocked,
  play: playShutter,
  toggle: toggleSound,
} = useShutterSound()

const offPhoto = on('photo', () => playShutter())

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

/** 跳過這一題：不收照片、不計分，直接下一題（最後一題就結算） */
const skipQuestion = () => {
  if (submittedCount.value > 0 && !window.confirm('已經有人交照片了，確定要跳過這一題嗎？')) return
  socket.skipRound()
}

/**
 * 房主邊拍邊排名次：拍照中按「投給它」照順序給名次，倒數照跑、大家照樣上傳。
 * 記照片主人而不是 photoId，對方重拍換照片時名次不會掉。
 */
const hostPicks = ref<string[]>([])
watch(
  () => question.value?.questionId,
  () => (hostPicks.value = []),
)
const pickLimit = computed(() => (maxWinners.value > 0 ? maxWinners.value : Infinity))
const pickRanks = computed(() =>
  Object.fromEntries(hostPicks.value.map((id, i) => [id, i + 1])) as Record<string, number>,
)
const togglePick = (playerId: string) => {
  const i = hostPicks.value.indexOf(playerId)
  if (i >= 0) hostPicks.value.splice(i, 1)
  else if (hostPicks.value.length < pickLimit.value) hostPicks.value.push(playerId)
}
/** 依名次排好的 photoId，給評選畫面與公布用 */
const pickedPhotoIds = computed(() =>
  hostPicks.value
    .map((pid) => roundPhotos.value.find((p) => p.playerId === pid)?.photoId)
    .filter((id): id is string => Boolean(id)),
)
/** 拍照中直接公布：收桌後馬上套用已投的名次（同一條連線依序處理，END_SHOOTING 一定先生效） */
const announceNow = () => {
  if (!pickedPhotoIds.value.length) return
  socket.endShooting()
  socket.pickWinners(pickedPhotoIds.value)
}
const closeLightbox = () => (lightboxIndex.value = -1)

const progress = computed(() => {
  if (!question.value?.timeLimit) return 0
  return Math.max(0, Math.min(1, timeLeft.value / question.value.timeLimit))
})

const urgent = computed(() => timeLeft.value <= 10)

const offRoundResult = on('roundResult', (result) => {
  if (result.winners.length) celebrate()
})

onBeforeUnmount(() => {
  offPhoto()
  offRoundResult()
})

const isLastQuestion = computed(
  () => !!question.value && question.value.questionNum >= question.value.totalQuestions,
)
</script>

<template>
  <main class="relative flex min-h-screen flex-col px-8 py-6 lg:px-12">
    <!-- 題目分類：放在畫面正上方中間，一眼看出這題是哪一類 -->
    <div
      v-if="question && !isPractice && question.category !== 'custom'"
      class="pointer-events-none absolute inset-x-0 top-3 flex justify-center"
    >
      <span class="rounded-full border border-white/15 bg-white/10 px-4 py-1 text-sm font-semibold text-slate-200 backdrop-blur">
        {{ CATEGORY_EMOJI[question.category] ?? '' }} {{ CATEGORY_LABELS[question.category] ?? question.category }}
      </span>
    </div>

    <!-- 題目列 -->
    <header class="flex shrink-0 flex-wrap items-center justify-between gap-6">
      <div class="min-w-0 flex-1">
        <button
          class="mb-1 text-xs text-slate-500 transition hover:text-white"
          @click="router.push(`/lobby/${roomId}`)"
        >
          ← 回大廳（遊戲會繼續）
        </button>
        <p class="text-sm text-slate-400">
          <span v-if="isPractice" class="rounded bg-amber-500/25 px-2 py-0.5 text-amber-200">
            試玩回合 · 不計分，大家都可以上傳
          </span>
          <template v-else>
            第 {{ question?.questionNum ?? '-' }} / {{ question?.totalQuestions ?? '-' }} 題
          </template>
          <span v-if="mode === 'group'" class="ml-2 rounded bg-blush-500/25 px-2 py-0.5">
            團體合體照
          </span>
        </p>
        <h1 class="mt-2 text-4xl font-black leading-tight lg:text-6xl">
          {{ question?.text ?? '準備中…' }}
        </h1>
      </div>

      <div class="flex shrink-0 items-center gap-6">
        <div v-if="status === 'shooting' && !isPractice" class="text-center">
          <p class="text-sm text-slate-400">已交卷</p>
          <p class="text-3xl font-black tabular-nums">
            {{ submittedCount }}<span class="text-lg text-slate-500">/{{ connectedCount }}</span>
          </p>
        </div>

        <div
          v-if="status === 'shooting' && !isPractice"
          class="relative grid h-28 w-28 place-items-center"
        >
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

        <button
          class="shrink-0 rounded-xl px-2 py-2 text-xl leading-none transition hover:bg-white/10"
          :title="soundOn ? '關閉快門音效' : '開啟快門音效'"
          :aria-label="soundOn ? '關閉快門音效' : '開啟快門音效'"
          @click="toggleSound()"
        >
          {{ soundOn ? '🔊' : '🔇' }}
        </button>

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

      <PhotoStrip
        v-if="isPractice"
        :photos="roundPhotos"
        :total="roundPhotos.length"
        title="試玩投稿（全部顯示）"
        :show-slots="false"
        @open="openLightbox"
      />
      <PhotoStrip
        v-else-if="voting"
        :photos="roundPhotos"
        :total="roundPhotos.length"
        :limit="voteCandidates"
        :title="`投票候選：湊滿 ${voteCandidates} 張開始 ${voteSeconds} 秒投票倒數`"
        hint="玩家拍照中也能先投 · 點照片看大圖"
        :vote-counts="voteCounts"
        @open="openLightbox"
      />
      <PhotoStrip
        v-else
        :photos="stripPhotos"
        :total="roundPhotos.length"
        :limit="stripLimit"
        hint="點照片看大圖 · 按「投給它」排名次，倒數照跑"
        :picks="pickRanks"
        :picks-full="hostPicks.length >= pickLimit"
        @open="openLightbox"
        @pick="togglePick"
      />

      <div class="flex items-center justify-between gap-4" :class="{ 'justify-end': isPractice }">
        <label v-if="!isPractice && !voting" class="flex items-center gap-2 text-xs text-slate-500">
          顯示最快
          <select
            v-model.number="stripLimit"
            class="rounded-lg border border-white/15 bg-slate-900 px-2 py-1 text-slate-200"
          >
            <option v-for="n in stripOptions" :key="n" :value="n">{{ n }}</option>
          </select>
          張
        </label>

        <button v-if="isPractice" class="btn-primary px-8 py-3" @click="socket.endPractice()">
          結束試玩，開始第 1 題 →
        </button>
        <div v-else class="flex items-center gap-3">
          <button class="btn-ghost px-6 py-3" @click="skipQuestion">
            {{ isLastQuestion ? '跳過這一題，直接結算 →' : '跳過這一題 →' }}
          </button>
          <button
            v-if="!voting && hostPicks.length"
            class="btn-primary px-8 py-3 text-lg"
            @click="announceNow"
          >
            公布結果（{{ hostPicks.length }} 位得獎）→
          </button>
          <!-- 投票模式：點照片只是看大圖，要提早開始投票得按這顆（不必等湊滿候選張數） -->
          <button
            v-if="voting"
            class="btn-primary px-8 py-3 text-lg"
            :disabled="!submittedCount"
            @click="socket.endShooting()"
          >
            開始投票 →
          </button>
        </div>
      </div>
    </section>

    <!-- 評選中 -->
    <!-- 投票倒數中 -->
    <section v-else-if="status === 'voting'" class="mt-6 flex min-h-0 flex-1 flex-col gap-5">
      <div class="flex items-end justify-between gap-6">
        <div>
          <h2 class="text-4xl font-black">🗳 投票中！</h2>
          <p class="mt-1 text-lg text-slate-400">拿出手機投給最喜歡的一張（不能投自己），已經投了 {{ totalVotes }} 票</p>
        </div>
        <p class="text-7xl font-black tabular-nums" :class="timeLeft <= 3 ? 'text-rose-400' : 'text-blush-300'">
          {{ timeLeft }}
        </p>
      </div>
      <TransitionGroup tag="ul" name="vote-rank" class="grid min-h-0 flex-1 auto-rows-min grid-cols-3 gap-4 lg:grid-cols-5">
        <li
          v-for="(photo, i) in ranked"
          :key="photo.playerId"
          class="overflow-hidden rounded-2xl border-4 bg-slate-900"
          :class="i === 0 && (voteCounts[photo.playerId] ?? 0) > 0 ? 'border-amber-300' : 'border-white/10'"
        >
          <img :src="photo.url" :alt="`${photo.playerName} 的投稿`" class="aspect-square w-full object-cover" />
          <div class="flex items-center justify-between gap-2 px-3 py-2">
            <span class="truncate font-semibold">{{ photo.playerAvatar }} {{ photo.playerName }}</span>
            <span class="shrink-0 text-2xl font-black text-blush-300">{{ voteCounts[photo.playerId] ?? 0 }} 票</span>
          </div>
        </li>
      </TransitionGroup>
      <div class="flex justify-end gap-3">
        <button class="btn-ghost px-6 py-3" @click="skipQuestion">
          {{ isLastQuestion ? '跳過這一題，直接結算 →' : '跳過這一題 →' }}
        </button>
        <button class="btn-primary px-8 py-3 text-lg" @click="socket.endVoting()">
          結束投票 →
        </button>
      </div>
    </section>

    <section v-else-if="status === 'judging'" class="mt-6 min-h-0 flex-1">
      <JudgePanel
        :photos="roundPhotos"
        :question-text="question?.text ?? ''"
        :max-winners="maxWinners"
        :initial-picked="pickedPhotoIds"
        @confirm="socket.pickWinners($event)"
        @skip="socket.skipRound()"
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
              {{ ['🥇', '🥈', '🥉'][winner.rank - 1] ?? `第 ${winner.rank} 名` }}
            </p>
            <p class="truncate font-bold">{{ winner.playerAvatar }} {{ winner.playerName }}</p>
            <p class="text-sm font-semibold text-blush-600">
              +{{ winner.points }} 分<span v-if="winner.votes"> · {{ winner.votes }} 票</span>
            </p>
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

      <!-- 自動下一題的倒數條：伺服器到點就換題，這條只是讓全場看得到還剩多久 -->
      <div
        v-if="resultSeconds > 0 && roundResult"
        :key="roundResult.questionIndex"
        class="fixed inset-x-0 bottom-0 z-40 h-2 bg-white/10"
      >
        <div
          class="auto-next-bar h-full bg-gradient-to-r from-blush-500 to-amber-300"
          :style="{ animationDuration: `${resultSeconds}s` }"
        />
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

    <!-- 其他狀態（例如還沒開始）也要有出路，避免變成沒有按鈕的死畫面 -->
    <section v-else class="mt-6 flex flex-1 flex-col items-center justify-center gap-4 text-center">
      <div class="text-7xl">⏳</div>
      <h2 class="text-2xl font-bold">這場還沒開始</h2>
      <p class="text-sm text-slate-400">回大廳按下開始，或確認大家都加入了沒。</p>
      <button class="btn-primary mt-2 px-8 py-3" @click="router.push(`/lobby/${roomId}`)">
        回大廳
      </button>
    </section>

    <!-- 瀏覽器在沒有互動過的頁面會擋掉音訊，給個一次性提示 -->
    <p
      v-if="soundOn && soundBlocked"
      class="pointer-events-none fixed bottom-4 left-1/2 z-30 -translate-x-1/2 rounded-full bg-amber-500/20 px-4 py-2 text-xs text-amber-200"
    >
      🔇 點畫面任一處啟用快門音效
    </p>

    <!-- 照片一到就彈出，停 3 秒淡出，新的直接蓋上舊的 -->
    <PhotoPopupLayer v-if="status === 'shooting'" />

    <!-- QR 放大：房間太大時給遠一點的人掃 -->
    <div
      v-if="qrExpanded"
      class="fixed inset-0 z-[60] flex flex-col items-center justify-center gap-6 bg-slate-950 p-8"
      @click="qrExpanded = false"
    >
      <!-- 刻意用不透明背景：放大 QR 時要把題目完全蓋住，不能讓還沒加入的人先看到題目 -->
      <QRCodeDisplay :value="shareUrl" :size="720" class="w-[min(56vh,64vw)]" />
      <p class="text-6xl font-black tracking-[0.2em] text-blush-300">{{ roomId }}</p>
      <!-- 連結區塊點了不要關掉放大畫面，方便選取或複製 -->
      <div class="flex w-full max-w-2xl items-center gap-3 rounded-2xl bg-white/5 p-2 pl-4" @click.stop>
        <span class="min-w-0 flex-1 select-all truncate font-mono text-lg text-slate-200">{{ shareUrl }}</span>
        <button class="btn-primary shrink-0 px-5 py-2.5" @click="copyShareUrl">
          {{ copied ? '✅ 已複製' : '複製連結' }}
        </button>
      </div>
      <p class="text-sm text-slate-600">點其他地方關閉</p>
    </div>

    <PhotoLightbox
      v-if="lightboxIndex >= 0"
      :photos="roundPhotos"
      :index="lightboxIndex"
      @update:index="lightboxIndex = $event"
      @close="closeLightbox"
    >
      <template #default="{ photo }">
      <!-- 正式題（非投票模式）拍照中：放大看也能直接投，倒數照跑 -->
      <div v-if="status === 'shooting' && !isPractice && !voting" class="w-full max-w-md">
        <button
          v-if="pickRanks[photo.playerId]"
          class="btn-ghost w-full py-5 text-xl"
          @click="togglePick(photo.playerId); closeLightbox()"
        >
          取消（目前第 {{ pickRanks[photo.playerId] }} 名）
        </button>
        <button
          v-else
          class="btn-primary w-full py-5 text-2xl"
          :disabled="hostPicks.length >= pickLimit"
          @click="togglePick(photo.playerId); closeLightbox()"
        >
          {{ hostPicks.length >= pickLimit ? '名額已滿' : `🗳 投給它（第 ${hostPicks.length + 1} 名）` }}
        </button>
      </div>
      <div v-else-if="isPractice" class="w-full max-w-md text-center">
        <p class="text-sm text-amber-300">試玩回合不計分，大家可以一直上傳，這裡沒有投票</p>
        <button class="btn-primary mt-3 w-full py-4 text-xl" @click="closeLightbox(); socket.endPractice()">
          結束試玩，開始第 1 題 →
        </button>
      </div>
      </template>
    </PhotoLightbox>
  </main>
</template>

<style scoped>
.auto-next-bar {
  width: 0;
  animation: auto-next linear forwards;
}
.vote-rank-move {
  transition: transform 0.4s ease;
}
@keyframes auto-next {
  to {
    width: 100%;
  }
}
</style>
