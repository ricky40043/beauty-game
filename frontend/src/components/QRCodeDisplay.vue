<script setup lang="ts">
import { ref, watchEffect } from 'vue'
import QRCode from 'qrcode'

const props = withDefaults(defineProps<{ value: string; size?: number }>(), { size: 320 })

const dataUrl = ref('')
const error = ref('')

watchEffect(async () => {
  if (!props.value) {
    dataUrl.value = ''
    return
  }

  try {
    dataUrl.value = await QRCode.toDataURL(props.value, {
      width: props.size,
      margin: 1,
      color: { dark: '#0f172a', light: '#ffffff' },
    })
    error.value = ''
  } catch {
    error.value = 'QR code 產生失敗'
  }
})
</script>

<template>
  <div class="inline-block rounded-3xl bg-white p-3 shadow-2xl">
    <img v-if="dataUrl" :src="dataUrl" alt="加入遊戲的 QR code" class="block h-auto w-full" />
    <div v-else class="grid aspect-square place-items-center px-6 text-center text-sm text-slate-500">
      {{ error || '產生中…' }}
    </div>
  </div>
</template>
