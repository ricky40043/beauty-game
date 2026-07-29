<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import QuestionPicker from '@/components/QuestionPicker.vue'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'
import type { RoomSetup } from '@/types'

const router = useRouter()
const socket = useSocketStore()
const ui = useUIStore()

const setup = reactive<RoomSetup>({
  hostName: '主持人',
  mode: 'solo',
  questionMode: 'random',
  totalQuestions: 10,
  questionTimeLimit: 60,
  difficulty: 'mixed',
  questionIds: [],
  customQuestions: [],
  requireNickname: false,
  practiceRound: true,
})

const submitting = ref(false)

const customCount = computed(() => setup.questionIds.length + setup.customQuestions.length)

const canSubmit = computed(() => {
  if (setup.questionMode === 'custom') return customCount.value > 0
  return setup.totalQuestions > 0
})

const create = () => {
  if (!canSubmit.value) {
    ui.showError('自選題目模式至少要選一題')
    return
  }
  submitting.value = true
  socket.createRoom(setup)

  // 後端回 ROOM_CREATED 就會自動導到大廳；沒回應時把按鈕解鎖讓使用者重試
  setTimeout(() => (submitting.value = false), 4000)
}
</script>

<template>
  <main class="mx-auto max-w-lg px-5 py-8">
    <button class="text-sm text-slate-400 hover:text-white" @click="router.push('/')">← 回首頁</button>
    <h1 class="mt-3 text-2xl font-black">開一間房</h1>
    <p class="mt-1 text-sm text-slate-400">這台裝置會變成主畫面，建議開在電視或筆電上。</p>

    <div class="mt-6 space-y-4">
      <!-- 模式 -->
      <section class="card">
        <h2 class="font-bold">遊戲模式</h2>
        <div class="mt-3 grid grid-cols-2 gap-3">
          <button
            class="rounded-2xl border p-4 text-left transition"
            :class="
              setup.mode === 'solo'
                ? 'border-blush-400 bg-blush-500/15'
                : 'border-white/10 bg-white/5 hover:bg-white/10'
            "
            @click="setup.mode = 'solo'"
          >
            <div class="text-2xl">🙋</div>
            <div class="mt-1 font-bold">單人</div>
            <p class="mt-1 text-xs leading-relaxed text-slate-400">
              每個人各自拍一張，比誰最會擺。
            </p>
          </button>

          <button
            class="rounded-2xl border p-4 text-left transition"
            :class="
              setup.mode === 'group'
                ? 'border-blush-400 bg-blush-500/15'
                : 'border-white/10 bg-white/5 hover:bg-white/10'
            "
            @click="setup.mode = 'group'"
          >
            <div class="text-2xl">👯</div>
            <div class="mt-1 font-bold">團體</div>
            <p class="mt-1 text-xs leading-relaxed text-slate-400">
              全場一起入鏡的合體照，得獎全員加分。
            </p>
          </button>
        </div>
      </section>

      <!-- 出題方式 -->
      <section class="card">
        <h2 class="font-bold">出題方式</h2>
        <div class="mt-3 flex gap-2">
          <button
            class="flex-1 rounded-2xl px-4 py-2.5 text-sm font-semibold transition"
            :class="
              setup.questionMode === 'random'
                ? 'bg-blush-500 text-white'
                : 'bg-white/5 text-slate-300 hover:bg-white/10'
            "
            @click="setup.questionMode = 'random'"
          >
            隨機抽題
          </button>
          <button
            class="flex-1 rounded-2xl px-4 py-2.5 text-sm font-semibold transition"
            :class="
              setup.questionMode === 'custom'
                ? 'bg-blush-500 text-white'
                : 'bg-white/5 text-slate-300 hover:bg-white/10'
            "
            @click="setup.questionMode = 'custom'"
          >
            自選題目與順序
          </button>
        </div>

        <div v-if="setup.questionMode === 'random'" class="mt-4 space-y-4">
          <label class="block">
            <span class="text-sm text-slate-300">題數：{{ setup.totalQuestions }} 題</span>
            <input
              v-model.number="setup.totalQuestions"
              type="range"
              min="1"
              max="30"
              class="mt-2 w-full accent-blush-500"
            />
          </label>

          <div class="flex gap-2">
            <button
              class="flex-1 rounded-xl px-3 py-2 text-xs font-medium transition"
              :class="
                setup.difficulty === 'basic'
                  ? 'bg-white/15 text-white'
                  : 'bg-white/5 text-slate-400 hover:bg-white/10'
              "
              @click="setup.difficulty = 'basic'"
            >
              只出基礎題
            </button>
            <button
              class="flex-1 rounded-xl px-3 py-2 text-xs font-medium transition"
              :class="
                setup.difficulty === 'mixed'
                  ? 'bg-white/15 text-white'
                  : 'bg-white/5 text-slate-400 hover:bg-white/10'
              "
              @click="setup.difficulty = 'mixed'"
            >
              基礎 + 進階混合
            </button>
          </div>
        </div>
      </section>

      <!-- 自選題目 -->
      <QuestionPicker
        v-if="setup.questionMode === 'custom'"
        :mode="setup.mode"
        :question-ids="setup.questionIds"
        :custom-questions="setup.customQuestions"
        @update:question-ids="setup.questionIds = $event"
        @update:custom-questions="setup.customQuestions = $event"
      />

      <!-- 其他設定 -->
      <section class="card space-y-4">
        <h2 class="font-bold">其他設定</h2>

        <label class="block">
          <span class="text-sm text-slate-300">每題拍照時間：{{ setup.questionTimeLimit }} 秒</span>
          <input
            v-model.number="setup.questionTimeLimit"
            type="range"
            min="15"
            max="180"
            step="5"
            class="mt-2 w-full accent-blush-500"
          />
        </label>

        <label class="block">
          <span class="text-sm text-slate-300">主持人名稱</span>
          <input v-model="setup.hostName" class="field mt-2 py-2 text-sm" maxlength="16" />
        </label>

        <button
          class="flex w-full items-center justify-between gap-4 rounded-2xl bg-slate-900/60 px-4 py-3 text-left"
          @click="setup.practiceRound = !setup.practiceRound"
        >
          <span>
            <span class="text-sm font-semibold">開場先試玩一輪</span>
            <span class="mt-0.5 block text-xs leading-relaxed text-slate-400">
              正式開始前先來一題不計分的暖身，大家都可以上傳、照片全部顯示，
              確認相機沒問題再由你按下開始
            </span>
          </span>
          <span
            class="relative h-6 w-11 shrink-0 rounded-full transition"
            :class="setup.practiceRound ? 'bg-blush-500' : 'bg-white/15'"
          >
            <span
              class="absolute top-0.5 h-5 w-5 rounded-full bg-white transition-all"
              :class="setup.practiceRound ? 'left-[22px]' : 'left-0.5'"
            />
          </span>
        </button>

        <button
          class="flex w-full items-center justify-between gap-4 rounded-2xl bg-slate-900/60 px-4 py-3 text-left"
          @click="setup.requireNickname = !setup.requireNickname"
        >
          <span>
            <span class="text-sm font-semibold">玩家需自行取名</span>
            <span class="mt-0.5 block text-xs leading-relaxed text-slate-400">
              關閉時掃碼直接入場，系統自動配可愛暱稱（可在大廳自己改）
            </span>
          </span>
          <span
            class="relative h-6 w-11 shrink-0 rounded-full transition"
            :class="setup.requireNickname ? 'bg-blush-500' : 'bg-white/15'"
          >
            <span
              class="absolute top-0.5 h-5 w-5 rounded-full bg-white transition-all"
              :class="setup.requireNickname ? 'left-[22px]' : 'left-0.5'"
            />
          </span>
        </button>
      </section>

      <button class="btn-primary w-full py-4 text-lg" :disabled="submitting" @click="create">
        {{ submitting ? '建立中…' : '建立房間' }}
      </button>
    </div>
  </main>
</template>
