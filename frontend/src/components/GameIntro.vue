<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'

/**
 * 開始遊戲前播在主畫面的玩法說明：一頁一頁自動翻，內容照房間設定產生
 * （拍照秒數、得獎名額、投票模式、試玩題目）。
 * 播放期間房間還在大廳，按「開始」才真的送 START_GAME，所以中途還是可以掃碼加入。
 */
const emit = defineEmits<{ start: []; close: [] }>()

const game = useGameStore()
const socket = useSocketStore()
const {
  mode,
  totalQuestions,
  questionTimeLimit,
  maxWinners,
  voteMode,
  voteCandidates,
  voteSeconds,
  practiceEnabled,
  practiceText,
} = storeToRefs(game)

type SlideKey = 'welcome' | 'shoot' | 'speed' | 'candidates' | 'judge' | 'vote' | 'score' | 'practice' | 'ready'

interface Slide {
  key: SlideKey
  kicker: string
  title: string
  lines: string[]
  chips: string[]
  /** 這一頁停幾秒自動翻下一頁 */
  seconds: number
}

const isGroup = computed(() => mode.value === 'group')

// 跟後端 game_service.go 的計分一致：得獎 100/80/60/40/20，第 6 名起 10 分
const WIN_POINTS = [100, 80, 60, 40, 20]
const ladder = computed(() =>
  WIN_POINTS.slice(0, maxWinners.value > 0 ? Math.min(maxWinners.value, WIN_POINTS.length) : WIN_POINTS.length),
)
const ladderHasMore = computed(() => maxWinners.value === 0 || maxWinners.value > WIN_POINTS.length)

const slides = computed<Slide[]>(() => {
  const list: Slide[] = []
  const groupBonus = '團體模式：這題有人得獎，全場每個人再加 50 分！'

  list.push({
    key: 'welcome',
    kicker: '怎麼玩？',
    title: '今日我最美',
    lines: [
      isGroup.value
        ? '看大螢幕的題目，全場一起入鏡拍合體照，比哪一張最美、最好笑！'
        : '看大螢幕的題目，用手機拍照上傳，比誰最美、最好笑！',
    ],
    chips: [`共 ${totalQuestions.value} 題`, isGroup.value ? '團體模式 · 合體照' : '單人模式 · 各拍各的'],
    seconds: 7,
  })

  list.push({
    key: 'shoot',
    kicker: '第一步',
    title: '題目出現，馬上拍！',
    lines: [
      `每一題有 ${questionTimeLimit.value} 秒可以拍。`,
      '按下快門就直接上傳，不滿意可以重拍，新的會蓋掉舊的。',
    ],
    chips: [],
    seconds: 8,
  })

  if (voteMode.value) {
    list.push({
      key: 'candidates',
      kicker: '手腳要快',
      title: `搶最快的前 ${voteCandidates.value} 張！`,
      lines: [
        `只有最快交的前 ${voteCandidates.value} 張照片能進入投票，湊滿就停止收照片。`,
        '有上傳就拿 10 分，最快的前三名再加 15、10、5 分。',
      ],
      chips: [],
      seconds: 9,
    })
    list.push({
      key: 'vote',
      kicker: '投票模式',
      title: '大家一起投票！',
      lines: [
        `湊滿 ${voteCandidates.value} 張或拍照時間到，就開始 ${voteSeconds.value} 秒的投票倒數。`,
        '在手機點你最喜歡的照片，按「投票給這個人」。',
      ],
      chips: ['每人 1 票', '可以改投', '不能投自己', '拍照中就能先投'],
      seconds: 10,
    })
    list.push({
      key: 'score',
      kicker: '怎麼算分',
      title: '票數越高，分數越高',
      lines: [
        maxWinners.value > 0 ? `票數前 ${maxWinners.value} 名得獎，同票時先交照片的人排前面。` : '有拿到票的都能排名，同票時先交照片的人排前面。',
        ...(isGroup.value ? [groupBonus] : []),
      ],
      chips: [],
      seconds: 8,
    })
  } else {
    list.push({
      key: 'speed',
      kicker: '手腳要快',
      title: '越快交，分數越高',
      lines: ['有上傳就拿 10 分，最快的前三名再加分！'],
      chips: [],
      seconds: 7,
    })
    list.push({
      key: 'judge',
      kicker: '怎麼算分',
      title: '主持人選出最美的',
      lines: [
        maxWinners.value > 0
          ? `時間到，主持人會選出前 ${maxWinners.value} 名。`
          : '時間到，主持人會幫所有照片排名次。',
        ...(isGroup.value ? [groupBonus] : []),
      ],
      chips: [],
      seconds: 8,
    })
  }

  if (practiceEnabled.value) {
    list.push({
      key: 'practice',
      kicker: '暖身',
      title: '先試玩一輪',
      lines: ['不計分、不倒數，先確認相機沒問題。', '主持人按下開始，才進正式的第 1 題。'],
      chips: [],
      seconds: 8,
    })
  }

  list.push({
    key: 'ready',
    kicker: '最後',
    title: '準備好了嗎？',
    lines: ['玩完所有題目，總分最高的人就是今日最美！'],
    chips: [],
    seconds: 6,
  })

  return list
})

const index = ref(0)
const paused = ref(false)
/** 翻頁方向，決定換頁動畫往哪邊滑 */
const forward = ref(true)

const slide = computed(() => slides.value[index.value])
const isLast = computed(() => index.value === slides.value.length - 1)
const startLabel = computed(() => (practiceEnabled.value ? '開始試玩 →' : '開始第 1 題 →'))

const go = (next: number) => {
  const target = Math.max(0, Math.min(slides.value.length - 1, next))
  if (target === index.value) return
  forward.value = target > index.value
  index.value = target
}

/** 頁首進度條跑完就自動翻頁；最後一頁停著等主持人按開始 */
const onBarEnd = () => {
  if (!isLast.value) go(index.value + 1)
}

// 把頁次轉播給玩家手機，手機會顯示「請看大螢幕」
watch(
  index,
  () => socket.gameIntro({ active: true, page: index.value, total: slides.value.length, title: slide.value.title }),
  { immediate: true },
)

const onKey = (e: KeyboardEvent) => {
  if (e.key === 'ArrowRight' || e.key === 'PageDown') go(index.value + 1)
  else if (e.key === 'ArrowLeft' || e.key === 'PageUp') go(index.value - 1)
  else if (e.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  // 按了開始的話伺服器會丟掉這則（房間已經不在大廳），手機由 GAME_STARTED 收掉說明
  socket.gameIntro({ active: false })
})

// ── 各頁插圖用的資料 ──────────────────────────────────────
const steps = computed(() => [
  { emoji: '📺', label: '看題目' },
  { emoji: isGroup.value ? '👯' : '📱', label: isGroup.value ? '一起入鏡拍照' : '拍照上傳' },
  { emoji: '🏆', label: '比誰最美' },
])

const fastest = [
  { face: '😎', bonus: 15, tilt: '-6deg' },
  { face: '🤪', bonus: 10, tilt: '4deg' },
  { face: '😘', bonus: 5, tilt: '-3deg' },
]

const faces = ['😎', '🤪', '😘', '🥳', '😜', '🤩', '😇', '🤓', '😏', '🥸']
/** 候選格子一張一張補滿的間隔，張數多就補快一點，整段動畫維持在 3 秒內 */
const slotGap = computed(() => Math.min(0.35, 3 / voteCandidates.value))

const voteCards = [
  { face: '🤪', votes: 2 },
  { face: '😎', votes: 5 },
  { face: '😘', votes: 1 },
]

const RING = 2 * Math.PI * 44
</script>

<template>
  <div
    class="stage-glow fixed inset-0 z-50 flex flex-col overflow-hidden px-6 py-5 lg:px-12"
    data-testid="game-intro"
  >
    <!-- 頁首：每頁一格的進度條，目前這頁跑完就自動翻頁 -->
    <div class="flex flex-none gap-2">
      <div v-for="(s, i) in slides" :key="s.key" class="h-1.5 flex-1 overflow-hidden rounded-full bg-white/15">
        <div v-if="i < index" class="h-full w-full bg-blush-400" />
        <div
          v-else-if="i === index"
          :key="`run-${index}`"
          class="intro-bar h-full bg-blush-400"
          :style="{ animationDuration: `${s.seconds}s`, animationPlayState: paused ? 'paused' : 'running' }"
          @animationend="onBarEnd"
        />
      </div>
    </div>

    <header class="mt-4 flex flex-none items-center justify-between gap-4">
      <button class="text-sm text-slate-400 transition hover:text-white" @click="emit('close')">✕ 回大廳</button>
      <p class="text-sm text-slate-400">
        玩法說明 <span class="tabular-nums text-slate-200">{{ index + 1 }} / {{ slides.length }}</span>
      </p>
      <button class="btn-ghost px-4 py-2 text-sm" @click="emit('start')">跳過說明，直接開始 →</button>
    </header>

    <!-- 內容 -->
    <div class="grid min-h-0 flex-1 place-items-center">
      <Transition :name="forward ? 'intro-next' : 'intro-prev'" mode="out-in">
        <section :key="slide.key" class="flex w-full flex-col items-center text-center">
          <!-- 插圖 -->
          <div class="flex h-[36vh] w-full items-center justify-center">
            <!-- 三個步驟 -->
            <div v-if="slide.key === 'welcome'" class="flex items-center justify-center gap-3 lg:gap-6">
              <template v-for="(step, i) in steps" :key="step.label">
                <div
                  class="intro-pop flex w-[24vh] flex-col items-center gap-3 rounded-3xl border border-white/10 bg-white/5 px-3 py-6"
                  :style="{ animationDelay: `${0.3 + i * 0.7}s` }"
                >
                  <span class="text-[9vh] leading-none">{{ step.emoji }}</span>
                  <span class="text-lg font-bold lg:text-2xl">{{ step.label }}</span>
                </div>
                <span
                  v-if="i < steps.length - 1"
                  class="intro-pop text-4xl font-black text-blush-300"
                  :style="{ animationDelay: `${0.65 + i * 0.7}s` }"
                >
                  →
                </span>
              </template>
            </div>

            <!-- 手機按快門 + 每題秒數 -->
            <div v-else-if="slide.key === 'shoot'" class="flex items-center justify-center gap-[7vh]">
              <div
                class="intro-pop relative h-[34vh] w-[18vh] overflow-hidden rounded-[3.2vh] border-4 border-white/30 bg-slate-900 shadow-2xl"
                :style="{ animationDelay: '0.2s' }"
              >
                <p class="absolute inset-x-0 top-[2.2vh] text-[1.7vh] text-slate-400">大螢幕出題</p>
                <div class="absolute inset-0 grid place-items-center text-[10vh]">{{ isGroup ? '👯' : '🤳' }}</div>
                <div
                  class="intro-shutter absolute bottom-[2.2vh] left-1/2 h-[5.5vh] w-[5.5vh] rounded-full border-4 border-white bg-blush-500"
                />
                <div class="intro-flash absolute inset-0 bg-white" />
                <p
                  class="intro-uploaded absolute inset-x-[1.5vh] top-[5.5vh] rounded-xl bg-emerald-500 py-[0.6vh] text-[1.9vh] font-bold text-white"
                >
                  ✓ 上傳成功
                </p>
              </div>

              <div class="intro-pop relative grid h-[24vh] w-[24vh] place-items-center" :style="{ animationDelay: '0.7s' }">
                <svg class="absolute inset-0 -rotate-90" viewBox="0 0 100 100">
                  <circle cx="50" cy="50" r="44" fill="none" stroke="rgba(255,255,255,0.1)" stroke-width="8" />
                  <circle
                    class="intro-ring"
                    cx="50"
                    cy="50"
                    r="44"
                    fill="none"
                    stroke="#f92e83"
                    stroke-width="8"
                    stroke-linecap="round"
                    :stroke-dasharray="RING"
                    :style="{ '--ring': RING }"
                  />
                </svg>
                <div class="leading-none">
                  <p class="text-[9vh] font-black tabular-nums" data-testid="intro-time-limit">{{ questionTimeLimit }}</p>
                  <p class="mt-1 text-[2.4vh] text-slate-400">秒 / 每題</p>
                </div>
              </div>
            </div>

            <!-- 最快前三名加分 -->
            <div v-else-if="slide.key === 'speed'" class="flex items-end justify-center gap-[4vh]">
              <div
                v-for="(card, i) in fastest"
                :key="card.face"
                class="intro-drop relative"
                :style="{ '--tilt': card.tilt, animationDelay: `${0.3 + i * 0.55}s` }"
              >
                <div class="h-[22vh] w-[17vh] rounded-2xl bg-white p-[1.1vh] pb-[3.6vh] shadow-2xl">
                  <div class="grid h-full place-items-center rounded-xl bg-gradient-to-br from-blush-200 to-indigo-200 text-[9vh]">
                    {{ card.face }}
                  </div>
                </div>
                <span
                  class="absolute -right-4 -top-4 rounded-full bg-amber-400 px-3 py-1 text-xl font-black text-slate-900 shadow-lg lg:text-2xl"
                >
                  +{{ card.bonus }}
                </span>
                <p class="mt-3 text-lg font-bold text-slate-300 lg:text-xl">第 {{ i + 1 }} 快</p>
              </div>
            </div>

            <!-- 投票模式：候選格子一張一張補滿 -->
            <div v-else-if="slide.key === 'candidates'" class="flex flex-col items-center gap-[3vh]">
              <div class="flex max-w-[86vw] flex-wrap items-center justify-center gap-[1.4vh]">
                <div
                  v-for="n in voteCandidates"
                  :key="n"
                  class="relative grid place-items-center rounded-2xl border-2 border-dashed border-white/20 font-bold text-slate-600"
                  :class="voteCandidates <= 10 ? 'h-[13vh] w-[10.5vh] text-[2.4vh]' : 'h-[9.5vh] w-[7.6vh] text-[2vh]'"
                >
                  {{ n }}
                  <div
                    class="intro-fill absolute inset-0 grid place-items-center rounded-2xl bg-white shadow-xl"
                    :class="voteCandidates <= 10 ? 'text-[6.5vh]' : 'text-[4.6vh]'"
                    :style="{ animationDelay: `${0.4 + (n - 1) * slotGap}s` }"
                  >
                    {{ faces[(n - 1) % faces.length] }}
                  </div>
                </div>
              </div>
              <p
                class="intro-pop rounded-full bg-blush-500 px-6 py-2 text-xl font-black shadow-lg shadow-blush-500/30 lg:text-2xl"
                :style="{ animationDelay: `${0.7 + voteCandidates * slotGap}s` }"
              >
                {{ voteCandidates }} 張額滿，開始投票！
              </p>
            </div>

            <!-- 投票：點照片 → 投票給這個人 + 投票秒數 -->
            <div v-else-if="slide.key === 'vote'" class="flex items-center justify-center gap-[6vh]">
              <div class="flex items-start gap-[2.6vh]">
                <div
                  v-for="(card, i) in voteCards"
                  :key="card.face"
                  class="intro-pop relative w-[14.5vh]"
                  :style="{ animationDelay: `${0.25 + i * 0.3}s` }"
                >
                  <div
                    class="h-[18vh] rounded-2xl bg-white p-[1vh] shadow-2xl"
                    :class="{ 'ring-4 ring-blush-400': i === 1 }"
                  >
                    <div class="grid h-full place-items-center rounded-xl bg-gradient-to-br from-blush-200 to-indigo-200 text-[8vh]">
                      {{ card.face }}
                    </div>
                  </div>
                  <span
                    class="absolute -right-3 -top-3 rounded-full bg-blush-500 px-2.5 py-1 text-lg font-black shadow-lg"
                    :class="{ 'intro-bump': i === 1 }"
                  >
                    🗳 {{ card.votes }}
                  </span>
                  <p
                    class="mt-[1.4vh] rounded-xl py-[0.9vh] text-[1.9vh] font-bold"
                    :class="i === 1 ? 'intro-tap bg-blush-500 text-white' : 'bg-white/10 text-slate-400'"
                  >
                    {{ i === 1 ? '投票給這個人' : '投這張' }}
                  </p>
                  <span v-if="i === 1" class="intro-finger absolute -bottom-[4.5vh] left-1/2 text-[5vh] leading-none">👆</span>
                </div>
              </div>

              <div class="intro-pop leading-none" :style="{ animationDelay: '1.2s' }">
                <p class="text-[13vh] font-black tabular-nums text-amber-300" data-testid="intro-vote-seconds">
                  {{ voteSeconds }}
                </p>
                <p class="mt-2 text-[2.6vh] font-bold text-slate-300">秒可以投票</p>
              </div>
            </div>

            <!-- 得獎分數階梯（主持人評選與投票共用） -->
            <div
              v-else-if="slide.key === 'judge' || slide.key === 'score'"
              class="flex h-full items-end justify-center gap-[2.2vh] pb-[1vh]"
            >
              <div v-for="(points, i) in ladder" :key="points" class="flex h-full w-[13vh] flex-col items-center justify-end">
                <p
                  class="intro-pop text-[4.4vh] font-black leading-none"
                  :class="i === 0 ? 'text-amber-300' : 'text-white'"
                  :style="{ animationDelay: `${0.75 + i * 0.3}s` }"
                >
                  {{ points }}<span class="ml-0.5 text-[2vh] font-bold text-slate-400">分</span>
                </p>
                <div
                  class="intro-grow mt-[1.2vh] w-full rounded-t-2xl bg-gradient-to-t"
                  :class="i === 0 ? 'from-amber-600 to-amber-300' : 'from-blush-700 to-blush-400'"
                  :style="{ height: `${(points / 100) * 62}%`, animationDelay: `${0.3 + i * 0.3}s` }"
                />
                <p class="mt-[1.2vh] text-[2.3vh] font-bold text-slate-300">第 {{ i + 1 }} 名</p>
              </div>
              <p
                v-if="ladderHasMore"
                class="intro-pop mb-[4.8vh] rounded-2xl border border-white/15 bg-white/5 px-4 py-3 text-[2.2vh] font-bold leading-snug text-slate-300"
                :style="{ animationDelay: `${0.75 + ladder.length * 0.3}s` }"
              >
                第 6 名起<br />都是 10 分
              </p>
            </div>

            <!-- 試玩題目 -->
            <div
              v-else-if="slide.key === 'practice'"
              class="intro-pop relative max-w-[80vw] rounded-[2.5rem] border border-amber-300/40 bg-amber-400/10 px-[7vh] py-[5vh]"
              :style="{ animationDelay: '0.2s' }"
            >
              <p class="text-[2.2vh] font-bold tracking-[0.3em] text-amber-200">試玩題目</p>
              <p class="mt-[1.5vh] text-[6vh] font-black leading-tight" data-testid="intro-practice-text">
                {{ practiceText }}
              </p>
              <span
                class="intro-stamp absolute -right-8 -top-6 rounded-xl border-4 border-rose-400 bg-slate-950/70 px-3 py-1 text-[2.8vh] font-black text-rose-400"
              >
                不計分
              </span>
            </div>

            <!-- 準備開始 -->
            <div v-else class="relative grid h-[30vh] w-[30vh] place-items-center">
              <span class="intro-crown text-[19vh] leading-none">👑</span>
              <span class="intro-twinkle absolute left-[1vh] top-[3vh] text-[5vh]">✨</span>
              <span class="intro-twinkle absolute right-0 top-[9vh] text-[4vh]" style="animation-delay: 0.5s">✨</span>
              <span class="intro-twinkle absolute bottom-[3vh] left-[5vh] text-[3.5vh]" style="animation-delay: 1s">✨</span>
            </div>
          </div>

          <!-- 文字 -->
          <p class="mt-[2.5vh] text-sm font-bold uppercase tracking-[0.3em] text-blush-300 lg:text-base">
            {{ slide.kicker }}
          </p>
          <h2 class="mt-2 text-4xl font-black leading-tight lg:text-6xl" data-testid="intro-title">{{ slide.title }}</h2>
          <div class="mt-[2vh] max-w-5xl space-y-1.5">
            <p
              v-for="(line, i) in slide.lines"
              :key="line"
              class="intro-rise text-xl leading-snug text-slate-300 lg:text-3xl"
              :style="{ animationDelay: `${0.25 + i * 0.25}s` }"
            >
              {{ line }}
            </p>
          </div>
          <div v-if="slide.chips.length" class="mt-[2vh] flex flex-wrap justify-center gap-2 lg:gap-3">
            <span
              v-for="(chip, i) in slide.chips"
              :key="chip"
              class="intro-pop rounded-full border border-white/15 bg-white/10 px-4 py-1.5 text-base font-semibold text-slate-100 lg:text-xl"
              :style="{ animationDelay: `${0.6 + i * 0.15}s` }"
            >
              {{ chip }}
            </span>
          </div>
        </section>
      </Transition>
    </div>

    <!-- 頁尾：上一頁／暫停／下一頁，最後一頁換成開始 -->
    <footer class="flex flex-none items-center justify-between gap-4">
      <button class="btn-ghost px-6 py-3" :disabled="index === 0" @click="go(index - 1)">← 上一頁</button>

      <button
        v-if="!isLast"
        class="rounded-xl px-3 py-2 text-sm text-slate-400 transition hover:bg-white/10 hover:text-white"
        @click="paused = !paused"
      >
        {{ paused ? '▶ 繼續自動翻頁' : '⏸ 暫停自動翻頁' }}
      </button>

      <button v-if="isLast" class="btn-primary px-10 py-4 text-2xl" data-testid="intro-start" @click="emit('start')">
        {{ startLabel }}
      </button>
      <button v-else class="btn-primary px-8 py-3 text-lg" @click="go(index + 1)">下一頁 →</button>
    </footer>
  </div>
</template>

<style scoped>
/* 頁首進度條：寬度從 0 跑到 100%，跑完觸發 animationend 翻頁 */
.intro-bar {
  animation: intro-bar linear forwards;
}
@keyframes intro-bar {
  from {
    width: 0;
  }
  to {
    width: 100%;
  }
}

/* 換頁：往前翻新的一頁從右邊滑進來，往回翻從左邊 */
.intro-next-enter-active,
.intro-prev-enter-active {
  transition: opacity 0.35s ease, transform 0.35s cubic-bezier(0.22, 1, 0.36, 1);
}
.intro-next-leave-active,
.intro-prev-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}
.intro-next-enter-from,
.intro-prev-leave-to {
  opacity: 0;
  transform: translateX(60px);
}
.intro-next-leave-to,
.intro-prev-enter-from {
  opacity: 0;
  transform: translateX(-60px);
}

/* 以下的進場動畫都用 both：延遲期間先維持起始（看不見）的樣子 */
.intro-pop {
  animation: intro-pop 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes intro-pop {
  0% {
    opacity: 0;
    transform: scale(0.6);
  }
  65% {
    opacity: 1;
    transform: scale(1.08);
  }
  100% {
    opacity: 1;
    transform: scale(1);
  }
}

.intro-rise {
  animation: intro-rise 0.5s ease-out both;
}
@keyframes intro-rise {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 照片從上面掉下來，落地後維持各自的傾斜角 */
.intro-drop {
  animation: intro-drop 0.6s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes intro-drop {
  0% {
    opacity: 0;
    transform: translateY(-60px) rotate(0deg) scale(0.8);
  }
  100% {
    opacity: 1;
    transform: translateY(0) rotate(var(--tilt)) scale(1);
  }
}

.intro-fill {
  animation: intro-pop 0.35s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.intro-grow {
  transform-origin: bottom;
  animation: intro-grow 0.6s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes intro-grow {
  from {
    transform: scaleY(0);
  }
  to {
    transform: scaleY(1);
  }
}

/* 手機：按快門 → 閃光 → 上傳成功，3 秒一輪 */
.intro-shutter {
  transform: translateX(-50%);
  animation: intro-shutter 3s ease-in-out 0.8s infinite;
}
@keyframes intro-shutter {
  0%,
  22%,
  34%,
  100% {
    transform: translateX(-50%) scale(1);
  }
  28% {
    transform: translateX(-50%) scale(0.78);
  }
}
.intro-flash {
  opacity: 0;
  animation: intro-flash 3s ease-out 0.8s infinite;
}
@keyframes intro-flash {
  0%,
  27%,
  45%,
  100% {
    opacity: 0;
  }
  30% {
    opacity: 0.9;
  }
}
.intro-uploaded {
  opacity: 0;
  animation: intro-uploaded 3s ease-out 0.8s infinite;
}
@keyframes intro-uploaded {
  0%,
  40% {
    opacity: 0;
    transform: translateY(-8px);
  }
  50%,
  88% {
    opacity: 1;
    transform: translateY(0);
  }
  100% {
    opacity: 0;
    transform: translateY(0);
  }
}

/* 倒數圈慢慢縮短 */
.intro-ring {
  animation: intro-ring 7s linear 1s both;
}
@keyframes intro-ring {
  from {
    stroke-dashoffset: 0;
  }
  to {
    stroke-dashoffset: calc(var(--ring) * 0.75px);
  }
}

/* 投票：手指點按鈕、票數跳一下 */
.intro-finger {
  animation: intro-finger 1.6s ease-in-out 1s infinite both;
}
@keyframes intro-finger {
  0%,
  100% {
    opacity: 1;
    transform: translate(-50%, 10px);
  }
  45% {
    opacity: 1;
    transform: translate(-50%, -8px);
  }
}
.intro-tap {
  animation: intro-tap 1.6s ease-in-out 1s infinite;
}
@keyframes intro-tap {
  0%,
  40%,
  60%,
  100% {
    transform: scale(1);
  }
  50% {
    transform: scale(0.92);
  }
}
.intro-bump {
  animation: intro-bump 1.6s ease-in-out 1s infinite;
}
@keyframes intro-bump {
  0%,
  50%,
  80%,
  100% {
    transform: scale(1);
  }
  62% {
    transform: scale(1.3);
  }
}

.intro-stamp {
  animation: intro-stamp 0.45s cubic-bezier(0.22, 1, 0.36, 1) 0.9s both;
}
@keyframes intro-stamp {
  from {
    opacity: 0;
    transform: rotate(12deg) scale(2.2);
  }
  to {
    opacity: 1;
    transform: rotate(12deg) scale(1);
  }
}

.intro-crown {
  animation: intro-crown 1.8s ease-in-out infinite;
}
@keyframes intro-crown {
  0%,
  100% {
    transform: translateY(0) rotate(-4deg);
  }
  50% {
    transform: translateY(-14px) rotate(4deg);
  }
}
.intro-twinkle {
  animation: intro-twinkle 1.5s ease-in-out infinite;
}
@keyframes intro-twinkle {
  0%,
  100% {
    opacity: 0.2;
    transform: scale(0.7);
  }
  50% {
    opacity: 1;
    transform: scale(1.15);
  }
}

@media (prefers-reduced-motion: reduce) {
  .intro-pop,
  .intro-rise,
  .intro-drop,
  .intro-fill,
  .intro-grow,
  .intro-stamp {
    animation-duration: 0.01s;
    animation-delay: 0s !important;
  }
}
</style>
