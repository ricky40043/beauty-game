<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { captureFromVideo, compressFile } from '@/utils/image'

const props = withDefaults(defineProps<{ disabled?: boolean; busy?: boolean }>(), {
  disabled: false,
  busy: false,
})

const emit = defineEmits<{ (e: 'submit', blob: Blob): void }>()

type Stage = 'starting' | 'live' | 'fallback' | 'preview'

const stage = ref<Stage>('starting')
const facing = ref<'user' | 'environment'>('user')
const videoEl = ref<HTMLVideoElement | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const hint = ref('')

const previewUrl = ref('')
const captured = ref<Blob | null>(null)

let stream: MediaStream | null = null

// 前鏡頭預覽鏡像，拍出來才跟使用者看到的一致
const mirrored = computed(() => facing.value === 'user')

const stopStream = () => {
  stream?.getTracks().forEach((track) => track.stop())
  stream = null
}

const startCamera = async () => {
  stopStream()

  // getUserMedia 只在 HTTPS 或 localhost 可用；區網 IP 走 http 會拿不到
  if (!navigator.mediaDevices?.getUserMedia) {
    stage.value = 'fallback'
    hint.value = '這個瀏覽器沒開放網頁相機，改用系統相機拍照'
    return
  }

  try {
    stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: facing.value, width: { ideal: 1280 }, height: { ideal: 1280 } },
      audio: false,
    })

    if (videoEl.value) {
      videoEl.value.srcObject = stream
      await videoEl.value.play()
    }
    stage.value = 'live'
    hint.value = ''
  } catch {
    stage.value = 'fallback'
    hint.value = '沒辦法開啟網頁相機（可能是權限或非 HTTPS），改用系統相機'
  }
}

const flipCamera = async () => {
  facing.value = facing.value === 'user' ? 'environment' : 'user'
  await startCamera()
}

const shoot = async () => {
  if (!videoEl.value || props.disabled) return

  const { blob, previewUrl: url } = await captureFromVideo(videoEl.value, mirrored.value)
  setPreview(blob, url)
}

const pickFile = (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  compressFile(file)
    .then(({ blob, previewUrl: url }) => setPreview(blob, url))
    .catch(() => (hint.value = '這張照片讀不出來，換一張試試'))
    .finally(() => (input.value = ''))
}

const setPreview = (blob: Blob, url: string) => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  captured.value = blob
  previewUrl.value = url
  stage.value = 'preview'
  stopStream()
}

const retake = async () => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = ''
  captured.value = null
  stage.value = 'starting'
  await startCamera()
}

const submit = () => {
  if (captured.value) emit('submit', captured.value)
}

onMounted(startCamera)

onBeforeUnmount(() => {
  stopStream()
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
})

defineExpose({ retake })
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- 拍照預覽 -->
    <div class="relative flex-1 overflow-hidden rounded-3xl bg-black">
      <video
        v-show="stage === 'live'"
        ref="videoEl"
        class="h-full w-full object-cover"
        :class="{ '-scale-x-100': mirrored }"
        playsinline
        muted
      />

      <img
        v-if="stage === 'preview'"
        :src="previewUrl"
        alt="剛拍的照片"
        class="h-full w-full object-cover"
      />

      <div
        v-if="stage === 'starting'"
        class="grid h-full place-items-center text-sm text-slate-400"
      >
        相機啟動中…
      </div>

      <div
        v-if="stage === 'fallback'"
        class="grid h-full place-items-center gap-3 px-6 text-center"
      >
        <div>
          <div class="text-5xl">📷</div>
          <p class="mt-3 text-sm leading-relaxed text-slate-300">{{ hint }}</p>
          <button class="btn-primary mt-4 px-6 py-3" @click="fileInput?.click()">
            開啟相機拍照
          </button>
        </div>
      </div>

      <button
        v-if="stage === 'live'"
        class="absolute right-3 top-3 rounded-full bg-black/50 px-3 py-2 text-sm backdrop-blur"
        aria-label="切換鏡頭"
        @click="flipCamera"
      >
        🔄 翻轉
      </button>
    </div>

    <!-- 操作區 -->
    <div class="mt-4 shrink-0">
      <div v-if="stage === 'preview'" class="flex gap-3">
        <button class="btn-ghost flex-1 py-4" :disabled="busy" @click="retake">重拍</button>
        <button class="btn-primary flex-[2] py-4 text-lg" :disabled="busy || disabled" @click="submit">
          {{ busy ? '上傳中…' : '上傳這張' }}
        </button>
      </div>

      <div v-else-if="stage === 'live'" class="flex justify-center">
        <button
          class="grid h-20 w-20 place-items-center rounded-full border-4 border-white/80 bg-white/15 backdrop-blur transition active:scale-90 disabled:opacity-40"
          :disabled="disabled"
          aria-label="拍照"
          @click="shoot"
        >
          <span class="h-14 w-14 rounded-full bg-white" />
        </button>
      </div>

      <p v-if="hint && stage !== 'fallback'" class="mt-2 text-center text-xs text-amber-300">
        {{ hint }}
      </p>
    </div>

    <input
      ref="fileInput"
      type="file"
      accept="image/*"
      capture="user"
      class="hidden"
      @change="pickFile"
    />
  </div>
</template>
