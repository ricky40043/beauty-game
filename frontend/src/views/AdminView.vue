<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  createQuestion,
  deleteExample,
  deleteQuestion,
  fetchAdminExamples,
  getAdminToken,
  setAdminToken,
  setQuestionDisabled,
  updateQuestion,
  uploadExample,
  type AdminQuestion,
} from '@/services/api'
import { useUIStore } from '@/stores/ui'
import { CATEGORY_LABELS } from '@/types'
import type { GameMode } from '@/types'

const router = useRouter()
const ui = useUIStore()

const questions = ref<AdminQuestion[]>([])
const storageDir = ref('')
const needsToken = ref(false)
const loading = ref(false)
const loadError = ref('')
const token = ref(getAdminToken())

const filter = ref<'group' | 'solo' | 'custom' | 'missing' | 'all'>('group')
const busyId = ref(0)

// 上傳後檔名不變，加一個版本參數才不會被瀏覽器快取擋住看到舊圖
const version = ref(Date.now())

const load = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const data = await fetchAdminExamples()
    questions.value = data.questions
    storageDir.value = data.storage
    needsToken.value = data.protected
    version.value = Date.now()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '載入失敗'
  } finally {
    loading.value = false
  }
}

onMounted(load)

const saveToken = () => {
  setAdminToken(token.value.trim())
  load()
}

const visible = computed(() => {
  switch (filter.value) {
    case 'group':
      return questions.value.filter((q) => q.mode === 'group')
    case 'solo':
      return questions.value.filter((q) => q.mode === 'solo')
    case 'custom':
      return questions.value.filter((q) => q.isCustom)
    case 'missing':
      return questions.value.filter((q) => !q.hasBuiltin && !q.hasCustom)
    default:
      return questions.value
  }
})

const counts = computed(() => ({
  custom: questions.value.filter((q) => q.isCustom).length,
  withImage: questions.value.filter((q) => q.hasBuiltin || q.hasCustom).length,
  disabled: questions.value.filter((q) => q.disabled).length,
}))

// ── 新增／編輯題目 ────────────────────────────────────────

const CATEGORY_OPTIONS = ['expression', 'imitate', 'pose', 'color', 'object', 'advanced', 'group', 'custom']

const editorOpen = ref(false)
const editingId = ref(0)
const draft = reactive({
  text: '',
  mode: 'solo' as GameMode,
  category: 'custom',
  difficulty: 1,
})

const openCreate = () => {
  editingId.value = 0
  draft.text = ''
  draft.mode = filter.value === 'group' ? 'group' : 'solo'
  draft.category = draft.mode === 'group' ? 'group' : 'custom'
  draft.difficulty = 1
  editorOpen.value = true
}

const openEdit = (question: AdminQuestion) => {
  editingId.value = question.id
  draft.text = question.text
  draft.mode = question.mode
  draft.category = question.category
  draft.difficulty = question.difficulty
  editorOpen.value = true
}

const saveDraft = async () => {
  if (!draft.text.trim()) {
    ui.showError('題目內容不能是空的')
    return
  }

  try {
    if (editingId.value) {
      await updateQuestion(editingId.value, {
        text: draft.text,
        category: draft.category,
        difficulty: draft.difficulty,
      })
      ui.showSuccess('題目已更新')
    } else {
      await createQuestion({ ...draft })
      ui.showSuccess('題目已新增')
    }
    editorOpen.value = false
    await load()
  } catch (error) {
    ui.showError(error instanceof Error ? error.message : '儲存失敗')
  }
}

const removeQuestion = async (question: AdminQuestion) => {
  if (!window.confirm(`確定要刪除「${question.text}」嗎？這題的示範圖也會一起刪掉。`)) return

  busyId.value = question.id
  try {
    await deleteQuestion(question.id)
    ui.showSuccess('題目已刪除')
    await load()
  } catch (error) {
    ui.showError(error instanceof Error ? error.message : '刪除失敗')
  } finally {
    busyId.value = 0
  }
}

const toggleDisabled = async (question: AdminQuestion) => {
  busyId.value = question.id
  try {
    await setQuestionDisabled(question.id, !question.disabled)
    ui.showSuccess(question.disabled ? '已重新啟用' : '已停用，之後不會被抽到')
    await load()
  } catch (error) {
    ui.showError(error instanceof Error ? error.message : '操作失敗')
  } finally {
    busyId.value = 0
  }
}

// ── 示範圖 ────────────────────────────────────────────────

const pick = async (question: AdminQuestion, event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return

  busyId.value = question.id
  try {
    await uploadExample(question.id, file)
    ui.showSuccess(`第 ${question.id} 題的圖已更新`)
    await load()
  } catch (error) {
    ui.showError(error instanceof Error ? error.message : '上傳失敗')
  } finally {
    busyId.value = 0
  }
}

const revert = async (question: AdminQuestion) => {
  busyId.value = question.id
  try {
    await deleteExample(question.id)
    ui.showSuccess(question.hasBuiltin ? '已還原成內建示範圖' : '已移除自訂圖')
    await load()
  } catch (error) {
    ui.showError(error instanceof Error ? error.message : '移除失敗')
  } finally {
    busyId.value = 0
  }
}

const imageSrc = (question: AdminQuestion) =>
  question.imageUrl ? `${question.imageUrl}?v=${version.value}` : ''
</script>

<template>
  <main class="mx-auto max-w-5xl px-5 py-8">
    <header class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <button class="text-sm text-slate-400 hover:text-white" @click="router.push('/')">
          ← 回首頁
        </button>
        <h1 class="mt-2 text-2xl font-black">題目與示範圖後台</h1>
        <p class="mt-1 text-sm text-slate-400">
          知道密碼的人都能在這裡新增題目、換示範圖。改動會立刻套用到之後開的房間。
        </p>
      </div>

      <div class="flex gap-2">
        <button class="btn-ghost px-4 py-2 text-sm" :disabled="loading" @click="load">
          {{ loading ? '載入中…' : '重新整理' }}
        </button>
        <button class="btn-primary px-4 py-2 text-sm" @click="openCreate">＋ 新增題目</button>
      </div>
    </header>

    <!-- 需要密碼時才顯示 -->
    <section v-if="needsToken || loadError" class="card mt-6">
      <h2 class="font-bold">後台密碼</h2>
      <p class="mt-1 text-xs text-slate-400">
        伺服器有設 ADMIN_TOKEN，要輸入才能新增題目或上傳圖片。密碼會存在這台裝置的瀏覽器裡。
      </p>
      <div class="mt-3 flex gap-2">
        <input
          v-model="token"
          type="password"
          class="field flex-1 py-2 text-sm"
          placeholder="輸入 ADMIN_TOKEN"
          @keyup.enter="saveToken"
        />
        <button class="btn-primary px-5 py-2 text-sm" @click="saveToken">套用</button>
      </div>
      <p v-if="loadError" class="mt-2 text-sm text-rose-300">{{ loadError }}</p>
    </section>

    <!-- 統計與篩選 -->
    <section class="card mt-6">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <p class="text-sm text-slate-300">
          共 <span class="font-bold">{{ questions.length }}</span> 題 ·
          自訂 <span class="font-bold text-blush-300">{{ counts.custom }}</span> 題 ·
          有圖 <span class="font-bold">{{ counts.withImage }}</span> 題 ·
          停用 <span class="font-bold text-slate-500">{{ counts.disabled }}</span> 題
        </p>

        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="option in [
              { key: 'group', label: '團體題' },
              { key: 'solo', label: '單人題' },
              { key: 'custom', label: '我新增的' },
              { key: 'missing', label: '還沒有圖' },
              { key: 'all', label: '全部' },
            ]"
            :key="option.key"
            class="rounded-full px-3 py-1 text-xs font-medium transition"
            :class="
              filter === option.key
                ? 'bg-blush-500 text-white'
                : 'bg-white/5 text-slate-300 hover:bg-white/10'
            "
            @click="filter = option.key as typeof filter"
          >
            {{ option.label }}
          </button>
        </div>
      </div>

      <p v-if="storageDir" class="mt-2 text-xs text-slate-500">
        題目與圖片存放位置：<code>{{ storageDir }}</code>
      </p>
    </section>

    <!-- 題目清單 -->
    <ul class="mt-6 grid gap-4 sm:grid-cols-2">
      <li
        v-for="question in visible"
        :key="question.id"
        class="card flex gap-4"
        :class="{ 'opacity-50': question.disabled }"
      >
        <div
          class="grid h-28 w-36 shrink-0 place-items-center overflow-hidden rounded-2xl bg-slate-900/70"
        >
          <img
            v-if="question.imageUrl"
            :src="imageSrc(question)"
            :alt="`第 ${question.id} 題的示範圖`"
            class="max-h-full max-w-full object-contain"
          />
          <span v-else class="text-xs text-slate-600">沒有圖</span>
        </div>

        <div class="flex min-w-0 flex-1 flex-col">
          <p class="flex flex-wrap items-center gap-1 text-xs text-slate-500">
            <span>#{{ question.id }}</span>
            <span class="rounded bg-white/5 px-1.5 py-0.5">
              {{ question.mode === 'group' ? '團體' : '單人' }}
            </span>
            <span class="rounded bg-white/5 px-1.5 py-0.5">
              {{ CATEGORY_LABELS[question.category] ?? question.category }}
            </span>
            <span v-if="question.difficulty === 2" class="rounded bg-white/5 px-1.5 py-0.5">
              進階
            </span>
            <span v-if="question.isCustom" class="rounded bg-blush-500/25 px-1.5 py-0.5 text-blush-200">
              自訂
            </span>
            <span v-if="question.disabled" class="rounded bg-rose-500/25 px-1.5 py-0.5 text-rose-200">
              已停用
            </span>
          </p>

          <p class="mt-1 flex-1 text-sm leading-snug">{{ question.text }}</p>

          <div class="mt-2 flex flex-wrap gap-1.5">
            <label
              class="btn-ghost cursor-pointer px-2.5 py-1.5 text-xs"
              :class="{ 'pointer-events-none opacity-50': busyId === question.id }"
            >
              {{ question.hasCustom ? '換圖' : '上傳圖' }}
              <input
                type="file"
                accept="image/png,image/jpeg,image/webp,image/svg+xml"
                class="hidden"
                @change="pick(question, $event)"
              />
            </label>

            <button
              v-if="question.hasCustom"
              class="btn-ghost px-2.5 py-1.5 text-xs"
              :disabled="busyId === question.id"
              @click="revert(question)"
            >
              {{ question.hasBuiltin ? '還原內建圖' : '移除圖' }}
            </button>

            <button
              v-if="question.isCustom"
              class="btn-ghost px-2.5 py-1.5 text-xs"
              :disabled="busyId === question.id"
              @click="openEdit(question)"
            >
              編輯
            </button>

            <button
              class="btn-ghost px-2.5 py-1.5 text-xs"
              :disabled="busyId === question.id"
              @click="toggleDisabled(question)"
            >
              {{ question.disabled ? '重新啟用' : '停用' }}
            </button>

            <button
              v-if="question.isCustom"
              class="btn-ghost px-2.5 py-1.5 text-xs text-rose-300"
              :disabled="busyId === question.id"
              @click="removeQuestion(question)"
            >
              刪除
            </button>
          </div>
        </div>
      </li>
    </ul>

    <p v-if="!loading && !visible.length" class="mt-8 text-center text-sm text-slate-400">
      這個篩選沒有題目。
    </p>

    <p class="mt-8 text-center text-xs leading-relaxed text-slate-500">
      內建題目是編譯在程式裡的，不能改也不能刪，但可以「停用」讓它不再被抽到。<br />
      你新增的題目與上傳的圖存在伺服器磁碟上。容器化部署若沒掛持久磁碟，重新部署會消失。
    </p>

    <!-- 新增／編輯題目 -->
    <div
      v-if="editorOpen"
      class="fixed inset-0 z-[70] flex items-center justify-center bg-slate-950/90 p-5 backdrop-blur"
      @click.self="editorOpen = false"
    >
      <div class="card w-full max-w-lg">
        <h2 class="text-xl font-bold">{{ editingId ? '編輯題目' : '新增題目' }}</h2>

        <label class="mt-4 block">
          <span class="text-sm text-slate-300">題目內容</span>
          <textarea
            v-model="draft.text"
            class="field mt-2 h-24 resize-none py-2 text-sm"
            maxlength="80"
            placeholder="例如：全體比出勝利手勢，一個都不能歪"
          />
          <span class="mt-1 block text-right text-xs text-slate-500">
            {{ draft.text.length }} / 80
          </span>
        </label>

        <div class="mt-2 grid gap-3 sm:grid-cols-3">
          <label class="block">
            <span class="text-sm text-slate-300">模式</span>
            <select
              v-model="draft.mode"
              class="field mt-2 py-2 text-sm"
              :disabled="Boolean(editingId)"
            >
              <option value="solo">單人</option>
              <option value="group">團體</option>
            </select>
          </label>

          <label class="block">
            <span class="text-sm text-slate-300">分類</span>
            <select v-model="draft.category" class="field mt-2 py-2 text-sm">
              <option v-for="c in CATEGORY_OPTIONS" :key="c" :value="c">
                {{ CATEGORY_LABELS[c] ?? c }}
              </option>
            </select>
          </label>

          <label class="block">
            <span class="text-sm text-slate-300">難度</span>
            <select v-model.number="draft.difficulty" class="field mt-2 py-2 text-sm">
              <option :value="1">基礎</option>
              <option :value="2">進階</option>
            </select>
          </label>
        </div>

        <p v-if="editingId" class="mt-2 text-xs text-slate-500">
          模式不能改 —— 改了的話已經排進舊房間的題目會對不上。要換模式請刪掉重新新增。
        </p>

        <div class="mt-5 flex gap-3">
          <button class="btn-ghost flex-1 py-3" @click="editorOpen = false">取消</button>
          <button class="btn-primary flex-[2] py-3" @click="saveDraft">
            {{ editingId ? '儲存' : '新增' }}
          </button>
        </div>
      </div>
    </div>
  </main>
</template>
