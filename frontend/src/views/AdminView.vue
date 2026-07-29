<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  deleteExample,
  fetchAdminExamples,
  getAdminToken,
  setAdminToken,
  uploadExample,
  type AdminQuestion,
} from '@/services/api'
import { useUIStore } from '@/stores/ui'
import { CATEGORY_LABELS } from '@/types'

const router = useRouter()
const ui = useUIStore()

const questions = ref<AdminQuestion[]>([])
const storageDir = ref('')
const needsToken = ref(false)
const loading = ref(false)
const loadError = ref('')
const token = ref(getAdminToken())

const filter = ref<'group' | 'missing' | 'all'>('group')
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
    case 'missing':
      return questions.value.filter((q) => !q.hasBuiltin && !q.hasCustom)
    default:
      return questions.value
  }
})

const counts = computed(() => ({
  custom: questions.value.filter((q) => q.hasCustom).length,
  builtin: questions.value.filter((q) => q.hasBuiltin && !q.hasCustom).length,
  none: questions.value.filter((q) => !q.hasBuiltin && !q.hasCustom).length,
}))

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
        <h1 class="mt-2 text-2xl font-black">題目示範圖後台</h1>
        <p class="mt-1 text-sm text-slate-400">
          內建的是程式畫的火柴人示範圖，你上傳的圖會蓋掉它，隨時可以還原。
        </p>
      </div>

      <button class="btn-ghost px-4 py-2 text-sm" :disabled="loading" @click="load">
        {{ loading ? '載入中…' : '重新整理' }}
      </button>
    </header>

    <!-- 需要密碼時才顯示 -->
    <section v-if="needsToken || loadError" class="card mt-6">
      <h2 class="font-bold">後台密碼</h2>
      <p class="mt-1 text-xs text-slate-400">
        伺服器有設 ADMIN_TOKEN，要輸入才能上傳或刪除。密碼會存在這台裝置的瀏覽器裡。
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
          自訂圖 <span class="font-bold text-blush-300">{{ counts.custom }}</span> 題 ·
          內建圖 <span class="font-bold">{{ counts.builtin }}</span> 題 ·
          沒有圖 <span class="font-bold text-slate-500">{{ counts.none }}</span> 題
        </p>

        <div class="flex gap-1.5">
          <button
            v-for="option in [
              { key: 'group', label: '團體題' },
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
        圖片存放位置：<code>{{ storageDir }}</code>
      </p>
    </section>

    <!-- 題目清單 -->
    <ul class="mt-6 grid gap-4 sm:grid-cols-2">
      <li v-for="question in visible" :key="question.id" class="card flex gap-4">
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
          <p class="text-xs text-slate-500">
            #{{ question.id }} ·
            {{ CATEGORY_LABELS[question.category] ?? question.category }}
            <span
              class="ml-1 rounded px-1.5 py-0.5"
              :class="
                question.hasCustom
                  ? 'bg-blush-500/25 text-blush-200'
                  : question.hasBuiltin
                    ? 'bg-white/10 text-slate-300'
                    : 'bg-white/5 text-slate-500'
              "
            >
              {{ question.hasCustom ? '自訂圖' : question.hasBuiltin ? '內建圖' : '無' }}
            </span>
          </p>

          <p class="mt-1 flex-1 text-sm leading-snug">{{ question.text }}</p>

          <div class="mt-2 flex gap-2">
            <label
              class="btn-ghost cursor-pointer px-3 py-1.5 text-xs"
              :class="{ 'pointer-events-none opacity-50': busyId === question.id }"
            >
              {{ question.hasCustom ? '換一張' : '上傳圖片' }}
              <input
                type="file"
                accept="image/png,image/jpeg,image/webp,image/svg+xml"
                class="hidden"
                @change="pick(question, $event)"
              />
            </label>

            <button
              v-if="question.hasCustom"
              class="btn-ghost px-3 py-1.5 text-xs text-rose-300"
              :disabled="busyId === question.id"
              @click="revert(question)"
            >
              {{ question.hasBuiltin ? '還原內建' : '移除' }}
            </button>
          </div>
        </div>
      </li>
    </ul>

    <p v-if="!loading && !visible.length" class="mt-8 text-center text-sm text-slate-400">
      這個篩選沒有題目。
    </p>

    <p class="mt-8 text-center text-xs leading-relaxed text-slate-500">
      提醒：容器化部署（Render / Cloud Run）的磁碟是暫時的，重新部署後自訂圖會消失，<br />
      需要長期保留請掛載持久磁碟或改接物件儲存。
    </p>
  </main>
</template>
