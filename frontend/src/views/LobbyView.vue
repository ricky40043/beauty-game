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

// 遊戲進行中時房主可能是從遊戲畫面按「回大廳」過來的，要留一條路回去
const gameInProgress = computed(() =>
  ['shooting', 'judging', 'round_result'].includes(status.value),
)

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
  <!--
    這一頁是投在電視或筆電上給全場看的，所以鎖成一個畫面、不能捲動 ——
    「開始遊戲」被推到摺線以下的話，房主得先滑一下才找得到。
    寬螢幕改成橫向雙欄（QR 在左、資訊與開始鍵在右）就塞得下了。
  -->
  <main v-if="isHost" class="flex h-[100dvh] flex-col overflow-hidden px-6 py-5 lg:px-12">
    <header class="flex flex-none flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl font-black lg:text-4xl">💄 今日我最美</h1>
        <p class="mt-1 text-sm text-slate-400">
          {{ mode === 'group' ? '團體模式 · 合體照' : '單人模式 · 各拍各的' }}
          · 共 {{ totalQuestions }} 題 · 每題 {{ questionTimeLimit }} 秒
        </p>
      </div>
      <button class="btn-ghost px-4 py-2 text-sm" @click="socket.leaveRoom()">關閉房間</button>
    </header>

    <!--
      刻意不顯示題目內容：把題目清單攤在大螢幕上等於先劇透。
      房主要確認順序請用「編輯題目」，那是個彈窗。
    -->
    <div
      class="mx-auto grid min-h-0 w-full max-w-6xl flex-1 gap-6 overflow-y-auto lg:grid-cols-[auto_1fr] lg:gap-12 lg:overflow-hidden"
    >
      <div class="flex items-center justify-center">
        <QRCodeDisplay :value="shareUrl" :size="520" class="w-[min(42vh,20rem)] lg:w-[min(58vh,24rem)]" />
      </div>

      <!--
        這裡不能用 grid 的 items-center 來置中：那會讓這一欄的高度由內容決定、
        不受列高限制，min-h-0 與底下玩家清單的 overflow-y-auto 就全部失效，
        人一多就從上下兩端溢出被裁掉（房號和「開始遊戲」都會不見）。
        改成讓欄位撐滿列高、再用 justify-center 置中，清單才縮得下來。
      -->
      <div class="flex min-h-0 flex-col items-center justify-center gap-4 lg:items-start">
        <div class="flex-none text-center lg:text-left">
          <p class="text-sm text-slate-400">房號</p>
          <p class="text-6xl font-black tracking-[0.2em] text-blush-300 lg:text-7xl">{{ roomId }}</p>
          <p class="mt-1 text-slate-400">掃碼或輸入房號加入，中途也能進來</p>
        </div>

        <button class="btn-ghost flex-none px-4 py-2 text-sm" @click="copyLink">複製加入連結</button>

        <!-- 已加入的玩家。人多的時候只讓這一塊自己捲，不要把開始鍵頂出畫面 -->
        <section class="flex min-h-0 w-full flex-col text-center lg:text-left">
          <p class="flex-none text-lg font-bold">
            已加入 <span class="text-blush-300">{{ players.length }}</span> 人
            <span v-if="!requireNickname" class="ml-2 text-xs font-normal text-slate-500">
              系統自動取名
            </span>
          </p>

          <p v-if="!players.length" class="mt-2 flex-none text-slate-400">等待玩家掃碼加入…</p>

          <ul
            v-else
            class="mt-2 flex min-h-0 flex-wrap justify-center gap-2 overflow-y-auto lg:justify-start"
          >
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
        </section>

        <!-- 只顯示題數，不顯示題目內容 -->
        <section
          class="flex flex-none flex-wrap items-center justify-center gap-2 text-sm text-slate-400 lg:justify-start"
        >
          <span>共 {{ totalQuestions }} 題 · 每題 {{ questionTimeLimit }} 秒</span>
          <button class="btn-ghost px-3 py-1.5 text-xs" @click="shuffleQuestions">重新隨機</button>
          <button class="btn-ghost px-3 py-1.5 text-xs" @click="openEditor">編輯題目</button>
        </section>

        <button
          v-if="gameInProgress"
          class="btn-primary w-full max-w-md flex-none py-4 text-xl"
          @click="$router.push(`/game/host/${roomId}`)"
        >
          回到進行中的遊戲 →
        </button>
        <button
          v-else
          class="btn-primary w-full max-w-md flex-none py-4 text-xl"
          :disabled="!players.length && !finishedLastGame"
          @click="start"
        >
          {{ finishedLastGame ? '開新的一局' : '開始遊戲' }}
        </button>
      </div>
    </div>

    <!-- 題目編輯 -->
    <div
      v-if="editingQuestions"
      class="fixed inset-0 z-50 overflow-y-auto bg-slate-950/90 p-5 backdrop-blur"
    >
      <div class="mx-auto max-w-lg space-y-4 pb-8">
        <header class="flex items-center justify-between">
          <div>
            <h2 class="text-xl font-bold">編輯題目與順序</h2>
            <p class="mt-0.5 text-xs text-amber-300">⚠️ 這裡會顯示題目內容，注意別讓玩家看到</p>
          </div>
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
