<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import ChampionReveal, { type Champion, type Highlight } from '@/components/ChampionReveal.vue'
import PhotoLightbox from '@/components/PhotoLightbox.vue'
import WinnersDownload from '@/components/WinnersDownload.vue'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'
import { celebrateFinale } from '@/utils/confetti'
import type { PhotoSubmission } from '@/types'

const game = useGameStore()
const socket = useSocketStore()
const router = useRouter()

const { isHost, roomId, scores, history, playerId } = storeToRefs(game)

/** 玩家回大廳等下一局：房間本來就還在，只是換個畫面看 */
const backToRoom = () => router.push(`/lobby/${roomId.value}`)

/** 關房是不可逆的，先確認一次 */
const confirmClose = () => {
  if (window.confirm('關閉房間會把所有人踢出去，照片也會全部刪除。確定要關嗎？')) {
    socket.leaveRoom()
  }
}

const lightboxIndex = ref(-1)

/** 同分並列第一的都算冠軍 */
const champions = computed<Champion[]>(() =>
  scores.value
    .filter((s) => s.rank === 1)
    .map((s) => ({
      playerId: s.playerId,
      playerName: s.playerName,
      avatar: s.avatar,
      score: s.score,
      photoUrl: bestPhotoOf(s.playerId),
    })),
)

/** 冠軍的代表作：名次最高的那張，同名次取比較早的題目 */
const bestPhotoOf = (id: string) => {
  let best: { url: string; rank: number } | undefined
  for (const round of history.value) {
    for (const w of round.winners) {
      if (w.playerId === id && (!best || w.rank < best.rank)) best = { url: w.url, rank: w.rank }
    }
  }
  return best?.url
}

/** 動畫要快閃的照片：每題的第一名，幾題就幾張（沒人得獎的題目跳過） */
const highlights = computed<Highlight[]>(() =>
  history.value.flatMap((round) => {
    const top = [...round.winners].sort((a, b) => a.rank - b.rank)[0]
    return top
      ? [
          {
            url: top.url,
            playerName: top.playerName,
            playerAvatar: top.playerAvatar,
            questionText: round.questionText,
            questionNum: round.questionIndex + 1,
          },
        ]
      : []
  }),
)

// 冠軍動畫只在主畫面播；玩家手機直接看結果頁最上面的冠軍與照片
const showReveal = ref(isHost.value && scores.value.length > 0)
const me = computed(() => scores.value.find((s) => s.playerId === playerId.value))

/** 把每題的得獎照片攤平成一面照片牆 */
const wall = computed<PhotoSubmission[]>(() =>
  history.value.flatMap((round) =>
    round.winners.map((winner) => ({
      photoId: winner.photoId,
      url: winner.url,
      roomId: game.roomId,
      questionIndex: round.questionIndex,
      playerId: winner.playerId,
      playerName: winner.playerName,
      playerAvatar: winner.playerAvatar,
      order: winner.rank,
      elapsedSec: 0,
      submittedAt: '',
    })),
  ),
)

const captionOf = (index: number) => {
  const photo = wall.value[index]
  const round = history.value.find((r) => r.questionIndex === photo.questionIndex)
  return round?.questionText ?? ''
}

onMounted(() => {
  if (scores.value.length && !showReveal.value) celebrateFinale()
})
</script>

<template>
  <main class="mx-auto min-h-screen max-w-5xl px-5 py-8">
    <header class="text-center">
      <div class="text-6xl">🏆</div>
      <h1 class="mt-3 text-3xl font-black lg:text-4xl">
        {{ champions.length > 1 ? '今日最美：並列冠軍' : '今日最美出爐' }}
      </h1>
      <div v-if="champions.length" class="mt-5 flex flex-wrap justify-center gap-5">
        <div v-for="champ in champions" :key="champ.playerId" class="flex flex-col items-center">
          <img
            v-if="champ.photoUrl"
            :src="champ.photoUrl"
            :alt="champ.playerName"
            class="aspect-square object-cover shadow-[0_0_40px_rgb(255_209_102/0.35)]"
            :class="[
              'rounded-2xl border-4 border-amber-300',
              champions.length > 1 ? 'w-36 sm:w-44' : 'w-56 sm:w-64',
            ]"
          />
          <p class="mt-3 text-xl">
            <span class="text-2xl">{{ champ.avatar }}</span>
            <span class="ml-2 font-black text-blush-300">{{ champ.playerName }}</span>
            <span class="ml-2 text-slate-400">{{ champ.score }} 分</span>
          </p>
        </div>
      </div>
    </header>

    <!-- 我的成績（玩家端） -->
    <section v-if="!isHost && me" class="card mt-6 text-center">
      <p class="text-sm text-slate-400">你的成績</p>
      <p class="mt-1 text-4xl font-black">第 {{ me.rank }} 名</p>
      <p class="mt-2 text-slate-300">
        {{ me.score }} 分 · 得獎 {{ me.wins }} 次 · 投稿 {{ me.uploads }} 張
      </p>
    </section>

    <!-- 排行榜 -->
    <section class="card mt-6">
      <h2 class="font-bold">最終排行榜</h2>
      <ol class="mt-4 space-y-2">
        <li
          v-for="score in scores"
          :key="score.playerId"
          class="flex items-center gap-3 rounded-2xl px-3 py-2.5"
          :class="[
            score.rank === 1 ? 'bg-blush-500/20' : 'bg-white/5',
            score.playerId === playerId ? 'ring-1 ring-blush-400/60' : '',
          ]"
        >
          <span class="w-8 text-center text-lg font-black text-slate-400">
            {{ ['🥇', '🥈', '🥉'][score.rank - 1] ?? score.rank }}
          </span>
          <span class="text-2xl">{{ score.avatar }}</span>
          <span class="flex-1 truncate font-semibold">{{ score.playerName }}</span>
          <span class="text-xs text-slate-400">得獎 {{ score.wins }} 次</span>
          <span class="w-16 text-right text-xl font-black tabular-nums">{{ score.score }}</span>
        </li>
      </ol>
    </section>

    <!-- 得獎照片牆 -->
    <section v-if="wall.length" class="card mt-6">
      <h2 class="font-bold">本場得獎照片牆</h2>
      <p class="mt-1 text-xs text-slate-500">照片只存在記憶體，離開這個房間後就會被刪掉。</p>
      <WinnersDownload v-if="isHost" :room-id="roomId" :items="highlights" class="mt-4 !justify-start" />

      <ul class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        <li v-for="(photo, index) in wall" :key="`${photo.photoId}-${index}`">
          <button
            class="block w-full overflow-hidden rounded-2xl border-2 border-white/15 transition hover:border-blush-400"
            @click="lightboxIndex = index"
          >
            <img :src="photo.url" :alt="photo.playerName" class="aspect-square w-full object-cover" />
            <span class="block truncate bg-black/60 px-2 py-1 text-[11px]">
              {{ photo.playerAvatar }} {{ photo.playerName }}
            </span>
          </button>
        </li>
      </ul>
    </section>

    <!-- 每題回顧 -->
    <section v-if="history.length" class="card mt-6">
      <h2 class="font-bold">每題回顧</h2>
      <ol class="mt-3 space-y-2 text-sm">
        <li
          v-for="round in history"
          :key="round.questionIndex"
          class="flex flex-wrap items-baseline gap-x-2 gap-y-1 rounded-xl bg-white/5 px-3 py-2"
        >
          <span class="font-bold text-slate-400">{{ round.questionIndex + 1 }}.</span>
          <span class="flex-1">{{ round.questionText }}</span>
          <span v-if="round.winners.length" class="text-blush-300">
            🥇 {{ round.winners[0].playerName }}
          </span>
          <span v-else class="text-slate-500">無人得獎</span>
        </li>
      </ol>
    </section>

    <div class="mt-8 flex flex-col gap-3 sm:flex-row">
      <!--
        房主按「離開房間」會真的刪掉整間房，所以文案要講清楚後果，
        並且跟「再來一局」在視覺上分開，避免當成返回鍵誤按。
      -->
      <template v-if="isHost">
        <button class="btn-primary flex-[2] py-4 text-lg" @click="socket.resetRoomToLobby()">
          🔄 再來一局（回到房間，同一批人）
        </button>
        <button class="btn-ghost flex-1 py-4 text-rose-300" @click="confirmClose">
          結束並關閉房間
        </button>
      </template>

      <template v-else>
        <button class="btn-primary flex-[2] py-4 text-lg" @click="backToRoom">
          🔄 回到房間等下一局
        </button>
        <button class="btn-ghost flex-1 py-4" @click="socket.leaveRoom()">離開房間</button>
      </template>
    </div>

    <PhotoLightbox
      v-if="lightboxIndex >= 0"
      :photos="wall"
      :index="lightboxIndex"
      @update:index="lightboxIndex = $event"
      @close="lightboxIndex = -1"
    />

    <p v-if="lightboxIndex >= 0" class="sr-only">{{ captionOf(lightboxIndex) }}</p>

    <ChampionReveal
      v-if="showReveal"
      :room-id="roomId"
      :highlights="highlights"
      :champions="champions"
      @done="showReveal = false"
    />
  </main>
</template>
