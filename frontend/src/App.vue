<script setup lang="ts">
import { onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'

const socket = useSocketStore()
const ui = useUIStore()
const { blockingDialog } = storeToRefs(socket)
const { toasts } = storeToRefs(ui)

onMounted(() => socket.connect())

const toastClass = (kind: string) =>
  ({
    success: 'border-emerald-400/40 bg-emerald-500/15 text-emerald-100',
    error: 'border-rose-400/40 bg-rose-500/15 text-rose-100',
    warning: 'border-amber-400/40 bg-amber-500/15 text-amber-100',
    info: 'border-sky-400/40 bg-sky-500/15 text-sky-100',
  })[kind] ?? 'border-white/20 bg-white/10'
</script>

<template>
  <div class="min-h-full stage-glow">
    <!--
      這裡刻意不包 <Transition mode="out-in">。
      遊戲的轉場常常是「兩則訊息連續觸發同一次跳頁」（例如 GAME_STARTED 緊接著
      NEW_QUESTION），離場動畫還沒跑完就被下一次導航打斷時，進場元件會卡在
      fade-enter-from 的 opacity:0，畫面整片空白。頁面轉場的價值遠低於這個風險。
    -->
    <RouterView />

    <!-- 提示訊息 -->
    <div
      class="pointer-events-none fixed inset-x-0 top-3 z-[70] flex flex-col items-center gap-2 px-4"
    >
      <TransitionGroup name="fade">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="pointer-events-auto max-w-sm rounded-2xl border px-4 py-2.5 text-sm font-medium shadow-lg backdrop-blur"
          :class="toastClass(toast.kind)"
          @click="ui.dismiss(toast.id)"
        >
          {{ toast.message }}
        </div>
      </TransitionGroup>
    </div>

    <!-- 房間關閉 / 連線中斷 -->
    <div
      v-if="blockingDialog"
      class="fixed inset-0 z-[80] flex items-center justify-center bg-slate-950/85 p-6 backdrop-blur"
    >
      <div class="card w-full max-w-sm text-center">
        <div class="text-4xl">😵</div>
        <h2 class="mt-3 text-xl font-bold">{{ blockingDialog.title }}</h2>
        <p class="mt-2 text-sm leading-relaxed text-slate-300">{{ blockingDialog.message }}</p>
        <button class="btn-primary mt-5 w-full" @click="socket.acknowledgeDialog()">確認</button>
      </div>
    </div>
  </div>
</template>
