<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { on } from '@/utils/bus'
import type { PhotoSubmission } from '@/types'

/** 照片彈出後停留多久才淡出 */
const HOLD_MS = 3000
/** 同時最多疊幾張，避免連續投稿時整片蓋滿 */
const MAX_STACK = 6

interface StackItem {
  key: number
  photo: PhotoSubmission
  tilt: number
  offsetX: number
  offsetY: number
  timer: number
}

const stack = ref<StackItem[]>([])
let nextKey = 1

/** 隨機 10~30 度，左右各半，像被隨手丟在桌上的相片 */
const randomTilt = () => {
  const degrees = 10 + Math.random() * 20
  return Math.random() < 0.5 ? -degrees : degrees
}

const drop = (key: number) => {
  const index = stack.value.findIndex((item) => item.key === key)
  if (index >= 0) {
    window.clearTimeout(stack.value[index].timer)
    stack.value.splice(index, 1)
  }
}

const push = (photo: PhotoSubmission) => {
  const key = nextKey++

  stack.value.push({
    key,
    photo,
    tilt: randomTilt(),
    // 位置也錯開一點，疊起來才像一疊相片而不是完全重合
    offsetX: Math.random() * 120 - 60,
    offsetY: Math.random() * 80 - 40,
    timer: window.setTimeout(() => drop(key), HOLD_MS),
  })

  // 超過上限就把最舊的先請走
  while (stack.value.length > MAX_STACK) {
    drop(stack.value[0].key)
  }
}

const off = on('photo', ({ photo, replaced }) => {
  // 重拍覆蓋不再彈一次，避免同一個人洗版
  if (!replaced) push(photo)
})

onBeforeUnmount(() => {
  off()
  stack.value.forEach((item) => window.clearTimeout(item.timer))
})
</script>

<template>
  <div class="pointer-events-none fixed inset-0 z-50 grid place-items-center overflow-hidden p-8">
    <TransitionGroup name="photo-pop">
      <!--
        每一張新照片都疊在最上面（z-index 依陣列順序遞增），
        停 3 秒後自己淡出，底下那幾張就露出來，像一疊翻動中的相片。
      -->
      <figure
        v-for="(item, index) in stack"
        :key="item.key"
        class="absolute w-[min(42vw,26rem)] rounded-sm bg-white p-3 pb-14 shadow-[0_30px_90px_rgba(0,0,0,0.7)]"
        :style="{
          zIndex: 50 + index,
          // 位移與旋轉走 CSS 變數，進退場的 keyframes 才能在縮放時保留角度
          '--tx': `${item.offsetX}px`,
          '--ty': `${item.offsetY}px`,
          '--tilt': `${item.tilt}deg`,
          transform: 'translate(var(--tx), var(--ty)) rotate(var(--tilt))',
        }"
      >
        <img
          :src="item.photo.url"
          :alt="`${item.photo.playerName} 的投稿`"
          class="block aspect-square w-full bg-slate-900 object-cover"
        />
        <figcaption
          class="absolute inset-x-3 bottom-3 flex items-center justify-between gap-2 text-slate-900"
        >
          <span class="flex min-w-0 items-center gap-1.5 font-bold">
            <span class="text-xl">{{ item.photo.playerAvatar }}</span>
            <span class="truncate">{{ item.photo.playerName }}</span>
          </span>
          <span class="shrink-0 text-sm font-semibold text-blush-600">
            #{{ item.photo.order }} · {{ item.photo.elapsedSec.toFixed(1) }}s
          </span>
        </figcaption>
      </figure>
    </TransitionGroup>
  </div>
</template>
