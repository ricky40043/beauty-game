<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useCheer } from '@/composables/useCheer'
import { useShutterSound } from '@/composables/useShutterSound'
import { celebrate, celebrateFinale, clearConfetti } from '@/utils/confetti'

export interface Highlight {
  url: string
  playerName: string
  playerAvatar: string
  questionText: string
}

export interface Champion {
  playerId: string
  playerName: string
  avatar: string
  score: number
  photoUrl?: string
}

const props = defineProps<{
  highlights: Highlight[]
  champions: Champion[]
}>()

const emit = defineEmits<{ done: [] }>()

type Phase = 'intro' | 'flash' | 'hush' | 'reveal'
const phase = ref<Phase>('intro')
/** 已經砸上畫面的照片張數 */
const shown = ref(0)
/** 冠軍登場的動畫播完，才把下載／取消按鈕放出來，不然會搶走揭曉的焦點 */
const showActions = ref(false)
const downloading = ref('')

const shutter = useShutterSound()
const cheer = useCheer()

const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false

// 每張照片固定一個傾斜角與些微偏移，疊起來才像一堆被丟上桌的拍立得
const tilts = props.highlights.map(() => ({
  tilt: `${(Math.random() * 24 - 12).toFixed(1)}deg`,
  tx: `${Math.round(Math.random() * 60 - 30)}px`,
  ty: `${Math.round(Math.random() * 40 - 20)}px`,
}))

const current = computed(() => props.highlights[shown.value - 1])

// 只留最上面幾張在 DOM 裡，題數多的時候不用一直疊一堆看不到的圖
const visible = computed(() =>
  props.highlights
    .map((photo, index) => ({ photo, index }))
    .slice(Math.max(0, shown.value - 6), shown.value),
)

/**
 * 第 i 張停留多久：從 650ms 一路加速到 220ms，像拉霸機轉到最後。
 * 題數很多時整段壓在 6 秒內，最快不低於 120ms，不然眼睛跟不上。
 */
const intervalOf = (i: number, n: number) => {
  const progress = n <= 1 ? 1 : i / (n - 1)
  const base = 650 - (650 - 220) * progress
  const naturalTotal = n * ((650 + 220) / 2)
  const scale = Math.min(1, 6000 / naturalTotal)
  return Math.max(120, base * scale)
}

const timers: number[] = []
const later = (ms: number, fn: () => void) => {
  timers.push(window.setTimeout(fn, ms))
}

const preload = (urls: string[]) =>
  Promise.race([
    Promise.all(
      urls.map(
        (url) =>
          new Promise<void>((resolve) => {
            const img = new Image()
            img.onload = img.onerror = () => resolve()
            img.src = url
          }),
      ),
    ),
    // 照片讀太慢就不等了，砸上去時再慢慢出來也比整段卡住好
    new Promise<void>((resolve) => setTimeout(resolve, 2500)),
  ])

const reveal = () => {
  phase.value = 'reveal'
  cheer.boom()
  cheer.fanfare()
  later(350, () => cheer.applause())
  celebrate()
  later(500, celebrateFinale)
  later(2200, () => (showActions.value = true))
}

const EXTENSIONS: Record<string, string> = {
  'image/png': 'png',
  'image/webp': 'webp',
  'image/gif': 'gif',
}

/**
 * 下載冠軍照片。照片 URL 跟頁面同源，但 <a download> 直接指向它時，
 * 伺服器回的 Content-Disposition 與副檔名不一定對，所以先抓成 blob 再自己命名。
 */
const download = async (champ: Champion) => {
  if (!champ.photoUrl || downloading.value) return
  downloading.value = champ.playerId
  try {
    const res = await fetch(champ.photoUrl)
    if (!res.ok) throw new Error(`下載失敗：${res.status}`)
    const blob = await res.blob()
    const href = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = href
    a.download = `今日最美冠軍-${champ.playerName}.${EXTENSIONS[blob.type] ?? 'jpg'}`
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(href), 10_000)
  } catch {
    // 抓不到 blob（例如照片已被清掉）就退回開新分頁，至少還能手動另存
    window.open(champ.photoUrl, '_blank')
  } finally {
    downloading.value = ''
  }
}

const runFlash = () => {
  const n = props.highlights.length
  if (!n || reducedMotion) {
    reveal()
    return
  }

  phase.value = 'flash'
  let at = 0
  for (let i = 0; i < n; i++) {
    later(at, () => {
      shown.value = i + 1
      shutter.play()
    })
    at += intervalOf(i, n)
  }

  // 最後一張停一下，然後全黑、靜音半秒，揭曉才有落差
  later(at + 250, () => (phase.value = 'hush'))
  later(at + 800, reveal)
}

let finished = false
const finish = () => {
  if (finished) return
  finished = true
  emit('done')
}

const skip = () => {
  timers.forEach(clearTimeout)
  timers.length = 0
  finish()
}

onMounted(async () => {
  clearConfetti()
  const urls = [
    ...props.highlights.map((h) => h.url),
    ...props.champions.flatMap((c) => (c.photoUrl ? [c.photoUrl] : [])),
  ]
  const started = Date.now()
  await preload(urls)
  if (finished) return
  // 開場標題至少給 1 秒，讓全場的視線先集中過來
  later(Math.max(0, 1000 - (Date.now() - started)), runFlash)
})

onBeforeUnmount(() => timers.forEach(clearTimeout))
</script>

<template>
  <div class="fixed inset-0 z-50 overflow-hidden bg-slate-950 text-white">
    <!-- 聚光燈 -->
    <div class="spotlight pointer-events-none absolute inset-0" :class="{ 'opacity-0': phase === 'hush' }" />

    <!-- 開場 -->
    <Transition name="fade">
      <div v-if="phase === 'intro'" class="absolute inset-0 flex flex-col items-center justify-center">
        <p class="text-sm tracking-[0.5em] text-blush-300">TODAY'S BEST</p>
        <h1 class="intro-title mt-3 text-5xl font-black lg:text-7xl">本場精彩回顧</h1>
      </div>
    </Transition>

    <!-- 快閃 -->
    <div v-if="phase === 'flash' || phase === 'hush'" class="absolute inset-0" :class="{ 'stack-out': phase === 'hush' }">
      <div class="absolute inset-0 flex items-center justify-center">
        <figure
          v-for="{ photo, index } in visible"
          :key="index"
          class="slam absolute w-[min(62vh,80vw)] rounded-xl bg-white p-3 pb-12 shadow-2xl"
          :style="{ '--tilt': tilts[index].tilt, '--tx': tilts[index].tx, '--ty': tilts[index].ty }"
        >
          <img :src="photo.url" :alt="photo.playerName" class="aspect-square w-full rounded-md object-cover" />
          <figcaption class="absolute inset-x-3 bottom-3 truncate text-center text-base font-bold text-slate-800">
            {{ photo.playerAvatar }} {{ photo.playerName }}
          </figcaption>
        </figure>
      </div>

      <!-- 每張落下時的閃光；用 key 讓動畫每張都重播 -->
      <div :key="shown" class="flash pointer-events-none absolute inset-0 bg-white" />

      <p
        v-if="current && phase === 'flash'"
        class="absolute inset-x-0 bottom-8 truncate px-8 text-center text-xl font-semibold text-white/80 lg:text-2xl"
      >
        <span class="mr-3 tabular-nums text-blush-300">{{ shown }} / {{ highlights.length }}</span>
        {{ current.questionText }}
      </p>
    </div>

    <!-- 揭曉 -->
    <div v-if="phase === 'reveal'" class="absolute inset-0 flex flex-col items-center justify-center px-8">
      <div class="rays pointer-events-none absolute left-1/2 top-1/2" />
      <span
        v-for="n in 26"
        :key="n"
        class="sparkle pointer-events-none absolute text-amber-200"
        :style="{
          left: `${(n * 37) % 100}%`,
          top: `${(n * 61) % 100}%`,
          animationDelay: `${(n % 9) * 0.17}s`,
          fontSize: `${12 + (n % 4) * 8}px`,
        }"
        >✦</span
      >

      <p class="reveal-label relative text-lg tracking-[0.4em] text-amber-200 lg:text-2xl">
        {{ champions.length > 1 ? '並列冠軍' : '今日最美' }}
      </p>

      <div class="relative mt-6 flex flex-wrap items-end justify-center gap-10">
        <div
          v-for="(champ, i) in champions"
          :key="champ.playerId"
          class="champ flex flex-col items-center"
          :style="{ animationDelay: `${0.15 + i * 0.18}s` }"
        >
          <div class="crown text-5xl lg:text-6xl">👑</div>
          <div
            class="gold-frame mt-2 overflow-hidden rounded-2xl"
            :class="champions.length > 1 ? 'w-[min(34vh,40vw)]' : 'w-[min(46vh,70vw)]'"
          >
            <img v-if="champ.photoUrl" :src="champ.photoUrl" :alt="champ.playerName" class="aspect-square w-full object-cover" />
            <div v-else class="flex aspect-square w-full items-center justify-center bg-slate-900 text-8xl">
              {{ champ.avatar }}
            </div>
          </div>
          <p class="gold-text mt-5 text-4xl font-black lg:text-6xl">{{ champ.avatar }} {{ champ.playerName }}</p>
          <p class="mt-2 text-xl text-amber-100/80 lg:text-2xl">{{ champ.score }} 分</p>
        </div>
      </div>
    </div>

    <!-- 頒獎完：下載冠軍照片或取消 -->
    <Transition name="rise">
      <div
        v-if="showActions"
        class="absolute inset-x-0 bottom-8 flex flex-wrap items-center justify-center gap-3 px-6"
      >
        <template v-for="champ in champions" :key="champ.playerId">
          <button
            v-if="champ.photoUrl"
            class="btn-primary px-6 py-3 text-lg"
            :disabled="!!downloading"
            @click="download(champ)"
          >
            {{ downloading === champ.playerId ? '下載中…' : '⬇ 下載冠軍照片' }}
            <span v-if="champions.length > 1" class="text-sm opacity-80">（{{ champ.playerName }}）</span>
          </button>
        </template>
        <button class="btn-ghost px-6 py-3 text-lg" @click="skip">取消</button>
      </div>
    </Transition>

    <button
      v-if="phase !== 'reveal'"
      class="absolute bottom-5 right-5 rounded-full bg-white/10 px-4 py-2 text-sm text-white/70 backdrop-blur transition hover:bg-white/20"
      @click="skip"
    >
      跳過
    </button>
  </div>
</template>

<style scoped>
.spotlight {
  background: radial-gradient(ellipse 55% 60% at 50% 45%, rgb(249 46 131 / 0.22), transparent 70%);
  transition: opacity 0.3s;
}

.fade-leave-active {
  transition: opacity 0.35s, transform 0.35s;
}
.fade-leave-to {
  opacity: 0;
  transform: scale(1.08);
}

.rise-enter-active {
  transition: opacity 0.5s, transform 0.5s cubic-bezier(0.22, 1, 0.36, 1);
}
.rise-enter-from {
  opacity: 0;
  transform: translateY(20px);
}

.intro-title {
  animation: rise 0.7s cubic-bezier(0.22, 1, 0.36, 1) both;
}

/* 位移與傾斜角走 CSS 變數，跟 PhotoPopupLayer 一樣，縮放才不會蓋掉旋轉 */
.slam {
  transform: translate(var(--tx), var(--ty)) rotate(var(--tilt));
  animation: slam 0.32s cubic-bezier(0.2, 1.4, 0.4, 1) both;
}

.flash {
  animation: flash 0.28s ease-out both;
}

.stack-out {
  animation: stack-out 0.45s ease-in forwards;
}

.rays {
  width: 220vmax;
  height: 220vmax;
  margin: -110vmax 0 0 -110vmax;
  background: repeating-conic-gradient(rgb(255 209 102 / 0.16) 0deg 9deg, transparent 9deg 22deg);
  mask-image: radial-gradient(circle, black 0%, transparent 45%);
  animation: spin 24s linear infinite, fade-in 0.8s ease-out both;
}

.sparkle {
  opacity: 0;
  animation: twinkle 1.6s ease-in-out infinite;
}

.reveal-label {
  animation: rise 0.6s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.champ {
  animation: champ-in 0.75s cubic-bezier(0.2, 1.3, 0.4, 1) both;
}

.crown {
  animation: bob 1.8s ease-in-out infinite;
}

.gold-frame {
  border: 6px solid #ffd166;
  box-shadow:
    0 0 0 3px rgb(255 255 255 / 0.25),
    0 0 60px 10px rgb(255 209 102 / 0.45);
}

.gold-text {
  background: linear-gradient(100deg, #ffd166 20%, #fff7d6 40%, #ffd166 60%);
  background-size: 200% 100%;
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  animation: shimmer 2.4s linear infinite;
}

@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(24px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes slam {
  from {
    opacity: 0;
    transform: translate(var(--tx), var(--ty)) rotate(var(--tilt)) scale(1.7);
  }
  to {
    opacity: 1;
    transform: translate(var(--tx), var(--ty)) rotate(var(--tilt)) scale(1);
  }
}

@keyframes flash {
  from {
    opacity: 0.3;
  }
  to {
    opacity: 0;
  }
}

@keyframes stack-out {
  to {
    opacity: 0;
    transform: scale(0.85);
    filter: blur(6px);
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes fade-in {
  from {
    opacity: 0;
  }
}

@keyframes twinkle {
  0%,
  100% {
    opacity: 0;
    transform: scale(0.4) rotate(0deg);
  }
  50% {
    opacity: 1;
    transform: scale(1) rotate(45deg);
  }
}

@keyframes champ-in {
  from {
    opacity: 0;
    transform: translateY(60px) scale(0.6);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes bob {
  50% {
    transform: translateY(-8px) rotate(-6deg);
  }
}

@keyframes shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .rays,
  .sparkle,
  .crown,
  .gold-text {
    animation: none;
  }
  .sparkle {
    opacity: 0.6;
  }
}
</style>
