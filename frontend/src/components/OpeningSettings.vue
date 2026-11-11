<script setup lang="ts">
import { computed, watch } from 'vue'
import { PRACTICE_PRESETS, type GameMode } from '@/types'

/** 開場設定：玩法說明、試玩一輪與試玩題目。建房與大廳共用 */
const props = defineProps<{ mode: GameMode }>()

const intro = defineModel<boolean>('intro', { required: true })
const practice = defineModel<boolean>('practice', { required: true })
/** 空字串 = 用這個模式的預設題 */
const practiceText = defineModel<string>('practiceText', { required: true })

const presets = computed(() => PRACTICE_PRESETS[props.mode])
const current = computed(() => practiceText.value.trim() || presets.value[0])
/** 自己打的題目；選的是現成選項時輸入框留空 */
const custom = computed(() => (presets.value.includes(current.value) ? '' : practiceText.value))

// 換模式時，另一個模式的現成題目不適用（單人的「比出小愛心」不該變成團體題），回到預設
watch(
  () => props.mode,
  () => {
    const others = Object.values(PRACTICE_PRESETS).flat()
    if (others.includes(practiceText.value) && !presets.value.includes(practiceText.value)) {
      practiceText.value = ''
    }
  },
)
</script>

<template>
  <div class="space-y-3">
    <button
      class="flex w-full items-center justify-between gap-4 rounded-2xl bg-slate-900/60 px-4 py-3 text-left"
      @click="intro = !intro"
    >
      <span>
        <span class="text-sm font-semibold">開始前先播玩法說明</span>
        <span class="mt-0.5 block text-xs leading-relaxed text-slate-400">
          按「開始遊戲」後主畫面先用動畫一頁一頁教大家怎麼玩，內容會照你的設定（拍照秒數、得獎名額、投票、試玩）自動產生，隨時可以跳過
        </span>
      </span>
      <span
        class="relative h-6 w-11 shrink-0 rounded-full transition"
        :class="intro ? 'bg-blush-500' : 'bg-white/15'"
      >
        <span
          class="absolute top-0.5 h-5 w-5 rounded-full bg-white transition-all"
          :class="intro ? 'left-[22px]' : 'left-0.5'"
        />
      </span>
    </button>

    <button
      class="flex w-full items-center justify-between gap-4 rounded-2xl bg-slate-900/60 px-4 py-3 text-left"
      @click="practice = !practice"
    >
      <span>
        <span class="text-sm font-semibold">開場先試玩一輪</span>
        <span class="mt-0.5 block text-xs leading-relaxed text-slate-400">
          正式開始前先來一題不計分的暖身，大家都可以上傳、照片全部顯示，確認相機沒問題再由你按下開始
        </span>
      </span>
      <span
        class="relative h-6 w-11 shrink-0 rounded-full transition"
        :class="practice ? 'bg-blush-500' : 'bg-white/15'"
      >
        <span
          class="absolute top-0.5 h-5 w-5 rounded-full bg-white transition-all"
          :class="practice ? 'left-[22px]' : 'left-0.5'"
        />
      </span>
    </button>

    <div v-if="practice" class="rounded-2xl bg-slate-900/60 px-4 py-3">
      <span class="text-sm text-slate-300">試玩題目</span>
      <span class="ml-2 text-xs text-slate-500">{{ current }}</span>
      <div class="mt-2 flex flex-wrap gap-2">
        <button
          v-for="text in presets"
          :key="text"
          class="rounded-full px-3 py-1.5 text-xs font-medium transition"
          :class="current === text ? 'bg-blush-500 text-white' : 'bg-white/5 text-slate-400 hover:bg-white/10'"
          @click="practiceText = text"
        >
          {{ text }}
        </button>
      </div>
      <input
        :value="custom"
        class="field mt-2 py-2 text-sm"
        maxlength="60"
        placeholder="或是自己打一題試玩題目"
        @input="practiceText = ($event.target as HTMLInputElement).value"
      />
    </div>
  </div>
</template>
