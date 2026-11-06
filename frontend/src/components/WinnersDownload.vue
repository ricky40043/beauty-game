<script setup lang="ts">
import { ref } from 'vue'
import { useUIStore } from '@/stores/ui'
import { buildCollage, saveBlob, type CollageItem } from '@/utils/collage'

/** 下載每題冠軍：zip（原始照片）或一張拼貼大圖。冠軍動畫結尾與結算頁共用 */
const props = defineProps<{ roomId: string; items: CollageItem[] }>()

const ui = useUIStore()
const making = ref(false)

const today = () =>
  new Date().toLocaleDateString('zh-TW', { year: 'numeric', month: '2-digit', day: '2-digit' })

const downloadCollage = async () => {
  if (making.value || !props.items.length) return
  making.value = true
  try {
    const blob = await buildCollage(props.items, '今日我最美 · 每題冠軍', `${today()} · 共 ${props.items.length} 題`)
    saveBlob(blob, `今日最美-${props.roomId}-拼貼.jpg`)
  } catch {
    ui.showError('拼貼圖產生失敗，可以改下載 Zip')
  } finally {
    making.value = false
  }
}
</script>

<template>
  <div v-if="items.length" class="flex flex-wrap items-center justify-center gap-3">
    <!-- zip 由伺服器直接串流，用一般連結下載就好，不必先抓進記憶體 -->
    <a class="btn-primary px-5 py-3" :href="`/api/rooms/${roomId}/winners.zip`" download>
      ⬇ 下載每題冠軍（Zip）
    </a>
    <button class="btn-primary px-5 py-3" :disabled="making" @click="downloadCollage">
      {{ making ? '產生中…' : '🖼 下載拼貼圖' }}
    </button>
    <slot />
  </div>
</template>
