<script setup lang="ts">
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import QRCodeDisplay from '@/components/QRCodeDisplay.vue'
import QuestionPicker from '@/components/QuestionPicker.vue'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'

const game = useGameStore()
const socket = useSocketStore()
const ui = useUIStore()

const {
  roomId,
  mode,
  status,
  isHost,
  players,
  playerName,
  avatar,
  totalQuestions,
  questionTimeLimit,
  requireNickname,
  joinUrl,
} = storeToRefs(game)

const editingQuestions = ref(false)
const draftQuestionIds = ref<number[]>([])
const draftCustomQuestions = ref<string[]>([])
const draftTimeLimit = ref(questionTimeLimit.value)

const renaming = ref(false)
const nameDraft = ref('')

const shareUrl = computed(() => joinUrl.value || `${window.location.origin}/join/${roomId.value}`)

const copyLink = async () => {
  try {
    await navigator.clipboard.writeText(shareUrl.value)
    ui.showSuccess('連結已複製')
  } catch {
    ui.showError('複製失敗，請手動複製網址')
  }
}

const openEditor = () => {
  draftQuestionIds.value = game.questions.filter((q) => !q.isCustom).map((q) => q.id)
  draftCustomQuestions.value = game.questions.filter((q) => q.isCustom).map((q) => q.text)
  draftTimeLimit.value = questionTimeLimit.value
  editingQuestions.value = true
}

const saveQuestions = () => {
  if (!draftQuestionIds.value.length && !draftCustomQuestions.value.length) {
    ui.showError('至少要留一題')
    return
  }
  socket.updateSettings({
    questionMode: 'custom',
    questionIds: draftQuestionIds.value,
    customQuestions: draftCustomQuestions.value,
    questionTimeLimit: draftTimeLimit.value,
  })
  editingQuestions.value = false
}

const shuffleQuestions = () => {
  socket.updateSettings({
    questionMode: 'random',
    totalQuestions: totalQuestions.value || 10,
    difficulty: 'mixed',
    questionTimeLimit: questionTimeLimit.value,
  })
}

// 上一場結束後房間狀態是 finished，這時候送 START_GAME 會被後端擋下來，
// 必須先重置回大廳，所以按鈕在這種情況下改成「開新的一局」。
const finishedLastGame = computed(() => status.value === 'finished')

const start = () => {
  if (finishedLastGame.value) {
    socket.resetRoomToLobby()
    return
  }
  if (!players.value.length) {
    ui.showError('等至少一位玩家加入再開始')
    return
  }
  socket.startGame()
}

const startRename = () => {
  nameDraft.value = playerName.value
  renaming.value = true
}

const confirmRename = () => {
  const name = nameDraft.value.trim()
  if (!name) return
  socket.rename(name)
  renaming.value = false
}
</script>

<template>
  <!-- ── 主畫面（房主）───────────────────────────────── -->
  <main v-if="isHost" class="min-h-screen px-6 py-8 lg:px-12">
    <header class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl font-black lg:text-4xl">💄 今日我最美</h1>
        <p class="mt-1 text-sm text-slate-400">
          {{ mode === 'group' ? '團體模式 · 合體照' : '單人模式 · 各拍各的' }}
          · 共 {{ totalQuestions }} 題 · 每題 {{ questionTimeLimit }} 秒
        </p>
      </div>
      <button class="btn-ghost px-4 py-2 text-sm" @click="socket.leaveRoom()">關閉房間</button>
    </header>

    <div class="mt-8 grid gap-8 lg:grid-cols-[auto_1fr]">
      <section class="flex flex-col items-center gap-4">
        <QRCodeDisplay :value="shareUrl" :size="360" class="w-64 lg:w-80" />
        <div class="text-center">
          <p class="text-sm text-slate-400">房號</p>
          <p class="text-5xl font-black tracking-[0.2em] text-blush-300">{{ roomId }}</p>
        </div>
        <button class="btn-ghost px-4 py-2 text-sm" @click="copyLink">複製加入連結</button>
      </section>

      <section class="flex flex-col gap-4">
        <div class="card flex-1">
          <header class="flex items-center justify-between">
            <h2 class="text-xl font-bold">
              已加入
              <span class="ml-1 text-blush-300">{{ players.length }}</span> 人
            </h2>
            <span v-if="!requireNickname" class="text-xs text-slate-400">系統自動取名模式</span>
          </header>

          <p v-if="!players.length" class="mt-6 text-center text-slate-400">
            等待玩家掃碼加入…
          </p>

          <ul v-else class="mt-4 flex flex-wrap gap-2">
            <li
              v-for="player in players"
              :key="player.id"
              class="flex items-center gap-2 rounded-2xl bg-white/5 px-3 py-2"
              :class="{ 'opacity-40': !player.isConnected }"
            >
              <span class="text-xl">{{ player.avatar }}</span>
              <span class="font-medium">{{ player.name }}</span>
            </li>
          </ul>
        </div>

        <div class="card">
          <header class="flex items-center justify-between">
            <h2 class="font-bold">題目清單（{{ totalQuestions }} 題）</h2>
            <div class="flex gap-2">
              <button class="btn-ghost px-3 py-1.5 text-xs" @click="shuffleQuestions">
                重新隨機
              </button>
              <button class="btn-ghost px-3 py-1.5 text-xs" @click="openEditor">編輯順序</button>
            </div>
          </header>

          <ol class="mt-3 max-h-40 space-y-1 overflow-y-auto pr-1 text-sm text-slate-300">
            <li v-for="(q, index) in game.questions" :key="`${q.id}-${index}`">
              {{ index + 1 }}. {{ q.text }}
            </li>
          </ol>
        </div>

        <button
          class="btn-primary w-full py-5 text-xl"
          :disabled="!players.length && !finishedLastGame"
          @click="start"
        >
          {{ finishedLastGame ? '開新的一局' : '開始遊戲' }}
        </button>
      </section>
    </div>

    <!-- 題目編輯 -->
    <div
      v-if="editingQuestions"
      class="fixed inset-0 z-50 overflow-y-auto bg-slate-950/90 p-5 backdrop-blur"
    >
      <div class="mx-auto max-w-lg space-y-4 pb-8">
        <header class="flex items-center justify-between">
          <h2 class="text-xl font-bold">編輯題目與順序</h2>
          <button class="text-sm text-slate-400" @click="editingQuestions = false">取消</button>
        </header>

        <label class="card block">
          <span class="text-sm text-slate-300">每題拍照時間：{{ draftTimeLimit }} 秒</span>
          <input
            v-model.number="draftTimeLimit"
            type="range"
            min="15"
            max="180"
            step="5"
            class="mt-2 w-full accent-blush-500"
          />
        </label>

        <QuestionPicker
          :mode="mode"
          :question-ids="draftQuestionIds"
          :custom-questions="draftCustomQuestions"
          @update:question-ids="draftQuestionIds = $event"
          @update:custom-questions="draftCustomQuestions = $event"
        />

        <button class="btn-primary w-full py-4" @click="saveQuestions">儲存</button>
      </div>
    </div>
  </main>

  <!-- ── 玩家手機端 ─────────────────────────────────── -->
  <main v-else class="mx-auto flex min-h-screen max-w-md flex-col gap-5 px-5 py-8">
    <header class="text-center">
      <p class="text-sm text-slate-400">房號 {{ roomId }}</p>
      <h1 class="mt-1 text-2xl font-black">等主持人開始…</h1>
      <p class="mt-2 text-sm text-slate-400">
        {{ mode === 'group' ? '團體模式：等等要一起入鏡拍合照' : '單人模式：等等各拍各的' }}
      </p>
    </header>

    <section class="card text-center">
      <div class="text-5xl">{{ avatar }}</div>
      <div v-if="!renaming" class="mt-2">
        <p class="text-xl font-bold">{{ playerName }}</p>
        <button class="mt-2 text-xs text-slate-400 underline underline-offset-2" @click="startRename">
          換個暱稱
        </button>
      </div>
      <div v-else class="mt-3 flex gap-2">
        <input v-model="nameDraft" class="field flex-1 py-2 text-sm" maxlength="16" />
        <button class="btn-primary px-4 py-2 text-sm" @click="confirmRename">確定</button>
      </div>
    </section>

    <section class="card">
      <h2 class="font-bold">房間內 {{ players.length }} 人</h2>
      <ul class="mt-3 flex flex-wrap gap-2">
        <li
          v-for="player in players"
          :key="player.id"
          class="flex items-center gap-1.5 rounded-xl bg-white/5 px-2.5 py-1.5 text-sm"
          :class="{ 'opacity-40': !player.isConnected }"
        >
          <span>{{ player.avatar }}</span>
          <span>{{ player.name }}</span>
        </li>
      </ul>
    </section>

    <section class="card text-sm leading-relaxed text-slate-300">
      <h2 class="font-bold text-white">等一下要做的事</h2>
      <p class="mt-2">
        題目會出現在畫面最上方，看到題目就馬上拍照上傳。越快上傳分數越高，被主持人選中還有大加分。
      </p>
    </section>

    <button class="btn-ghost mt-auto w-full py-3 text-sm" @click="socket.leaveRoom()">
      離開房間
    </button>
  </main>
</template>
