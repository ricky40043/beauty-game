<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { fetchQuestionBank } from '@/services/api'
import { CATEGORY_LABELS } from '@/types'
import type { GameMode, Question } from '@/types'

const props = defineProps<{
  mode: GameMode
  questionIds: number[]
  customQuestions: string[]
}>()

const emit = defineEmits<{
  (e: 'update:questionIds', value: number[]): void
  (e: 'update:customQuestions', value: string[]): void
}>()

const bank = ref<Question[]>([])
const loading = ref(false)
const loadError = ref('')
const keyword = ref('')
const activeCategory = ref('all')
const customDraft = ref('')

// 預覽示範圖：不先問後端有沒有圖，直接載入、失敗就顯示提示
const previewing = ref<Question | null>(null)
const previewFailed = ref(false)

const openPreview = (question: Question) => {
  previewFailed.value = false
  previewing.value = question
}

const loadBank = async () => {
  loading.value = true
  loadError.value = ''
  try {
    bank.value = await fetchQuestionBank(props.mode)
  } catch {
    loadError.value = '題庫載入失敗，請確認後端有啟動'
  } finally {
    loading.value = false
  }
}

onMounted(loadBank)

// 切換模式時題庫整批換掉，已選的題目也要清空（solo / group 的題目不通用）
watch(
  () => props.mode,
  () => {
    emit('update:questionIds', [])
    loadBank()
  },
)

const byId = computed(() => new Map(bank.value.map((q) => [q.id, q])))

const categories = computed(() => {
  const set = new Set(bank.value.map((q) => q.category))
  return ['all', ...set]
})

const filtered = computed(() => {
  const kw = keyword.value.trim()
  return bank.value.filter((q) => {
    if (activeCategory.value !== 'all' && q.category !== activeCategory.value) return false
    if (kw && !q.text.includes(kw)) return false
    return true
  })
})

const selectedSet = computed(() => new Set(props.questionIds))
const totalPicked = computed(() => props.questionIds.length + props.customQuestions.length)

const toggle = (id: number) => {
  const next = [...props.questionIds]
  const index = next.indexOf(id)
  if (index >= 0) next.splice(index, 1)
  else next.push(id)
  emit('update:questionIds', next)
}

const move = (index: number, delta: number) => {
  const next = [...props.questionIds]
  const target = index + delta
  if (target < 0 || target >= next.length) return
  ;[next[index], next[target]] = [next[target], next[index]]
  emit('update:questionIds', next)
}

const removeAt = (index: number) => {
  const next = [...props.questionIds]
  next.splice(index, 1)
  emit('update:questionIds', next)
}

const addAllFiltered = () => {
  const next = [...props.questionIds]
  filtered.value.forEach((q) => {
    if (!next.includes(q.id)) next.push(q.id)
  })
  emit('update:questionIds', next)
}

const clearAll = () => {
  emit('update:questionIds', [])
  emit('update:customQuestions', [])
}

const addCustom = () => {
  const text = customDraft.value.trim()
  if (!text) return
  emit('update:customQuestions', [...props.customQuestions, text.slice(0, 60)])
  customDraft.value = ''
}

const removeCustom = (index: number) => {
  const next = [...props.customQuestions]
  next.splice(index, 1)
  emit('update:customQuestions', next)
}

const label = (category: string) =>
  category === 'all' ? '全部' : (CATEGORY_LABELS[category] ?? category)
</script>

<template>
  <div class="space-y-4">
    <!-- 已選清單：順序就是出題順序 -->
    <section class="card">
      <header class="flex items-center justify-between">
        <h3 class="font-bold">
          出題順序
          <span class="ml-1 text-sm font-normal text-slate-400">共 {{ totalPicked }} 題</span>
        </h3>
        <button
          v-if="totalPicked"
          class="text-xs text-slate-400 underline underline-offset-2 hover:text-rose-300"
          @click="clearAll"
        >
          全部清除
        </button>
      </header>

      <p v-if="!totalPicked" class="mt-3 text-sm text-slate-400">
        還沒選題目。從下面的題庫點選，或自己新增一題。
      </p>

      <ol v-else class="mt-3 space-y-2">
        <li
          v-for="(id, index) in questionIds"
          :key="id"
          class="flex items-center gap-2 rounded-2xl bg-slate-900/60 px-3 py-2"
        >
          <span class="w-6 shrink-0 text-center text-sm font-bold text-blush-300">
            {{ index + 1 }}
          </span>
          <span class="flex-1 text-sm leading-snug">{{ byId.get(id)?.text ?? '（題目已失效）' }}</span>
          <div class="flex shrink-0 gap-1">
            <button
              v-if="byId.get(id)"
              class="rounded-lg bg-white/5 px-2 py-1 text-xs"
              aria-label="看示範圖"
              @click="openPreview(byId.get(id)!)"
            >
              🖼
            </button>
            <button
              class="rounded-lg bg-white/5 px-2 py-1 text-xs disabled:opacity-30"
              :disabled="index === 0"
              aria-label="上移"
              @click="move(index, -1)"
            >
              ↑
            </button>
            <button
              class="rounded-lg bg-white/5 px-2 py-1 text-xs disabled:opacity-30"
              :disabled="index === questionIds.length - 1"
              aria-label="下移"
              @click="move(index, 1)"
            >
              ↓
            </button>
            <button
              class="rounded-lg bg-rose-500/15 px-2 py-1 text-xs text-rose-300"
              aria-label="移除"
              @click="removeAt(index)"
            >
              ✕
            </button>
          </div>
        </li>

        <li
          v-for="(text, index) in customQuestions"
          :key="`custom-${index}`"
          class="flex items-center gap-2 rounded-2xl bg-blush-500/10 px-3 py-2"
        >
          <span class="w-6 shrink-0 text-center text-sm font-bold text-blush-300">
            {{ questionIds.length + index + 1 }}
          </span>
          <span class="flex-1 text-sm leading-snug">
            {{ text }}
            <span class="ml-1 rounded bg-blush-500/25 px-1.5 py-0.5 text-[10px]">自訂</span>
          </span>
          <button
            class="shrink-0 rounded-lg bg-rose-500/15 px-2 py-1 text-xs text-rose-300"
            aria-label="移除"
            @click="removeCustom(index)"
          >
            ✕
          </button>
        </li>
      </ol>

      <div class="mt-3 flex gap-2">
        <input
          v-model="customDraft"
          class="field flex-1 py-2 text-sm"
          maxlength="60"
          placeholder="自己出一題…"
          @keyup.enter="addCustom"
        />
        <button class="btn-ghost shrink-0 px-4 py-2 text-sm" @click="addCustom">新增</button>
      </div>
      <p class="mt-1 text-xs text-slate-500">自訂題目會排在題庫題目後面。</p>
    </section>

    <!-- 題庫 -->
    <section class="card">
      <header class="flex items-center justify-between gap-2">
        <h3 class="font-bold">題庫</h3>
        <button
          v-if="filtered.length"
          class="text-xs text-slate-400 underline underline-offset-2 hover:text-blush-300"
          @click="addAllFiltered"
        >
          加入目前篩選的 {{ filtered.length }} 題
        </button>
      </header>

      <input v-model="keyword" class="field mt-3 py-2 text-sm" placeholder="搜尋題目關鍵字…" />

      <div class="mt-3 flex flex-wrap gap-1.5">
        <button
          v-for="category in categories"
          :key="category"
          class="rounded-full px-3 py-1 text-xs font-medium transition"
          :class="
            activeCategory === category
              ? 'bg-blush-500 text-white'
              : 'bg-white/5 text-slate-300 hover:bg-white/10'
          "
          @click="activeCategory = category"
        >
          {{ label(category) }}
        </button>
      </div>

      <p v-if="loading" class="mt-4 text-sm text-slate-400">載入題庫中…</p>
      <p v-else-if="loadError" class="mt-4 text-sm text-rose-300">{{ loadError }}</p>
      <p v-else-if="!filtered.length" class="mt-4 text-sm text-slate-400">沒有符合的題目。</p>

      <ul v-else class="mt-3 max-h-80 space-y-1.5 overflow-y-auto pr-1">
        <li v-for="q in filtered" :key="q.id">
          <button
            class="flex w-full items-center gap-3 rounded-2xl px-3 py-2 text-left transition"
            :class="
              selectedSet.has(q.id) ? 'bg-blush-500/20 ring-1 ring-blush-400/50' : 'hover:bg-white/5'
            "
            @click="toggle(q.id)"
          >
            <span
              class="grid h-5 w-5 shrink-0 place-items-center rounded-md border text-[11px]"
              :class="
                selectedSet.has(q.id)
                  ? 'border-blush-400 bg-blush-500 text-white'
                  : 'border-white/25'
              "
            >
              {{ selectedSet.has(q.id) ? '✓' : '' }}
            </span>
            <span class="flex-1 text-sm leading-snug">{{ q.text }}</span>
            <span class="shrink-0 rounded bg-white/5 px-1.5 py-0.5 text-[10px] text-slate-400">
              {{ CATEGORY_LABELS[q.category] ?? q.category }}{{ q.difficulty === 2 ? ' · 進階' : '' }}
            </span>
            <span
              class="shrink-0 rounded-lg bg-white/5 px-2 py-1 text-xs hover:bg-white/15"
              role="button"
              aria-label="看示範圖"
              @click.stop="openPreview(q)"
            >
              🖼
            </span>
          </button>
        </li>
      </ul>
    </section>

    <!-- 示範圖預覽 -->
    <div
      v-if="previewing"
      class="fixed inset-0 z-[70] flex flex-col items-center justify-center gap-4 bg-slate-950/95 p-6 backdrop-blur"
      @click.self="previewing = null"
    >
      <p class="max-w-xl text-center text-xl font-bold">{{ previewing.text }}</p>

      <img
        v-show="!previewFailed"
        :src="`/api/questions/${previewing.id}/example`"
        :alt="`${previewing.text} 的示範圖`"
        class="max-h-[60vh] max-w-full rounded-2xl bg-slate-900/60 p-2"
        @error="previewFailed = true"
      />

      <p v-if="previewFailed" class="max-w-sm text-center text-sm leading-relaxed text-slate-400">
        這題還沒有示範圖。<br />
        可以到「題目示範圖後台」自己上傳一張。
      </p>

      <button class="btn-ghost px-6 py-2" @click="previewing = null">關閉</button>
    </div>
  </div>
</template>
