<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchRoom } from '@/services/api'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'

const props = defineProps<{ roomId?: string }>()

const router = useRouter()
const route = useRoute()
const socket = useSocketStore()
const ui = useUIStore()

const code = ref((props.roomId ?? '').toUpperCase())
const nickname = ref('')
const roomInfo = ref<{ hostName: string; mode: string; requireNickname: boolean } | null>(null)
const checking = ref(false)
const scanning = ref(false)

let scanner: { stop: () => Promise<void>; clear: () => void } | null = null

/**
 * 掃 QR code（或點加入連結）進來、房間又不用取名時，直接幫他加入，不必再按一次。
 * 手動打房號不自動加入：打錯一碼就可能跑進別人的房間。
 * 被「沒有身分」導過來的（from=orphan）也不自動，可能是主持人自己的舊分頁。
 */
const autoJoin = ref(Boolean(props.roomId) && route.query.from !== 'orphan')
const autoJoining = ref(false)
const needNickname = computed(() => Boolean(roomInfo.value?.requireNickname))

watch(
  [roomInfo, () => socket.isConnected],
  ([info, connected]) => {
    if (!autoJoin.value || !info || info.requireNickname || !connected) return
    autoJoin.value = false
    autoJoining.value = true
    socket.joinRoom(code.value, '')
    // 正常會直接跳到大廳；萬一伺服器沒回應，把按鈕還給使用者
    setTimeout(() => (autoJoining.value = false), 5000)
  },
  { immediate: true },
)

/** 房號滿 6 碼就先問後端這房存不存在，順便知道要不要取名 */
const checkRoom = async () => {
  if (code.value.length < 6) {
    roomInfo.value = null
    return
  }

  checking.value = true
  try {
    const info = await fetchRoom(code.value)
    roomInfo.value = info
    if (!info) ui.showError('找不到這個房間')
  } finally {
    checking.value = false
  }
}

watch(code, (value) => {
  code.value = value.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 6)
  if (code.value.length === 6) checkRoom()
  else roomInfo.value = null
})

onMounted(() => {
  if (code.value.length === 6) checkRoom()
})

const join = () => {
  if (code.value.length !== 6) {
    ui.showError('請輸入 6 碼房號')
    return
  }
  if (roomInfo.value?.requireNickname && !nickname.value.trim()) {
    ui.showError('這個房間需要輸入暱稱')
    return
  }
  socket.joinRoom(code.value, nickname.value.trim())
}

// ── QR 掃描 ────────────────────────────────────────────
const startScan = async () => {
  scanning.value = true
  try {
    const { Html5Qrcode } = await import('html5-qrcode')
    const instance = new Html5Qrcode('qr-reader')
    scanner = instance

    await instance.start(
      { facingMode: 'environment' },
      { fps: 10, qrbox: { width: 220, height: 220 } },
      (decoded: string) => {
        const matched = decoded.match(/\/join\/([A-Z0-9]{6})/i) ?? decoded.match(/^([A-Z0-9]{6})$/i)
        if (matched) {
          autoJoin.value = true
          code.value = matched[1].toUpperCase()
          stopScan()
        }
      },
      () => {
        // 每一幀沒掃到都會呼叫，忽略即可
      },
    )
  } catch {
    scanning.value = false
    ui.showError('無法開啟相機掃描，請改用手動輸入房號')
  }
}

const stopScan = async () => {
  if (scanner) {
    try {
      await scanner.stop()
      scanner.clear()
    } catch {
      // 已經停掉了
    }
    scanner = null
  }
  scanning.value = false
}

onBeforeUnmount(stopScan)
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-md flex-col gap-5 px-5 py-8">
    <button class="self-start text-sm text-slate-400 hover:text-white" @click="router.push('/')">
      ← 回首頁
    </button>

    <h1 class="text-2xl font-black">加入遊戲</h1>

    <div v-show="scanning" class="card">
      <div id="qr-reader" class="overflow-hidden rounded-2xl" />
      <button class="btn-ghost mt-3 w-full py-2 text-sm" @click="stopScan">取消掃描</button>
    </div>

    <button v-if="!scanning" class="btn-ghost w-full py-3" @click="startScan">
      📷 掃描 QR code
    </button>

    <section class="card">
      <label class="block">
        <span class="text-sm text-slate-300">房號</span>
        <input
          v-model="code"
          class="field mt-2 text-center text-3xl font-black tracking-[0.4em]"
          placeholder="ABC123"
          autocapitalize="characters"
          autocomplete="off"
          maxlength="6"
        />
      </label>

      <p v-if="checking" class="mt-2 text-xs text-slate-400">查詢房間中…</p>
      <p v-else-if="roomInfo" class="mt-2 text-xs text-emerald-300">
        找到房間：{{ roomInfo.hostName }} 的{{ roomInfo.mode === 'group' ? '團體' : '單人' }}場
      </p>

      <label v-if="roomInfo?.requireNickname" class="mt-4 block">
        <span class="text-sm text-slate-300">你的暱稱</span>
        <input
          v-model="nickname"
          class="field mt-2"
          maxlength="16"
          placeholder="輸入暱稱"
          enterkeyhint="done"
          @keydown.enter.prevent="join"
        />
      </label>

      <p v-else-if="roomInfo" class="mt-4 rounded-2xl bg-slate-900/60 px-4 py-3 text-xs leading-relaxed text-slate-400">
        這個房間不用取名，系統會幫你配一個可愛暱稱，進大廳後可以自己改。
      </p>

      <button
        class="btn-primary mt-5 w-full py-4 text-lg"
        :disabled="code.length !== 6 || autoJoining"
        @click="join"
      >
        {{ autoJoining ? '正在幫你加入…' : needNickname ? '確定' : '加入遊戲' }}
      </button>
    </section>
  </main>
</template>
