import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

const STORAGE_KEY = 'beauty_sound_enabled'
const SOUND_URL = '/sounds/shutter.mp3'

/**
 * 主畫面收到照片時播的快門聲。
 *
 * 用 Web Audio 而不是 <audio>：需求是「上一聲還在播就停掉、重新播一次」，
 * 而 HTMLAudioElement 的 pause() 會讓前一個 play() 的 Promise 以 AbortError
 * 被拒絕，連續投稿時要一直去分辨哪個錯誤該忽略。Web Audio 每次播放都是獨立
 * 的 source node，停掉舊的、開一個新的，語意乾淨也沒有解碼延遲。
 */
export function useShutterSound() {
  const enabled = ref(localStorage.getItem(STORAGE_KEY) !== 'false')
  /** 瀏覽器因為還沒有使用者互動而擋掉播放 */
  const blocked = ref(false)

  let ctx: AudioContext | null = null
  let buffer: AudioBuffer | null = null
  let current: AudioBufferSourceNode | null = null

  watch(enabled, (value) => localStorage.setItem(STORAGE_KEY, String(value)))

  const load = async () => {
    const AudioCtx = window.AudioContext ?? (window as any).webkitAudioContext
    if (!AudioCtx) return

    try {
      ctx = new AudioCtx()
      const res = await fetch(SOUND_URL)
      if (!res.ok) throw new Error(`載入音效失敗：${res.status}`)
      buffer = await ctx.decodeAudioData(await res.arrayBuffer())

      // 自動播放政策：沒有使用者互動過的頁面，AudioContext 會是 suspended。
      // 房主按「開始遊戲」就算互動，但直接重新整理進遊戲畫面時不算，
      // 所以補一個一次性的監聽，任何點擊或按鍵都能解鎖。
      if (ctx.state === 'suspended') {
        blocked.value = true
        const unlock = () => {
          ctx?.resume().then(() => (blocked.value = false))
          window.removeEventListener('pointerdown', unlock)
          window.removeEventListener('keydown', unlock)
        }
        window.addEventListener('pointerdown', unlock, { once: true })
        window.addEventListener('keydown', unlock, { once: true })
      }
    } catch {
      // 音效只是加分項，失敗就安靜地不播，不要影響遊戲
      ctx = null
      buffer = null
    }
  }

  /** 播一次快門聲。正在播的那一聲會被停掉，從頭重播。 */
  const play = () => {
    if (!enabled.value || !ctx || !buffer) return

    if (ctx.state === 'suspended') {
      blocked.value = true
      return
    }

    // 停掉上一聲：連拍時每一張都聽得到「喀」，而不是被前一聲蓋過去
    if (current) {
      try {
        current.stop()
      } catch {
        // 已經自然播完的 node 再 stop 會拋錯，忽略即可
      }
      current = null
    }

    const source = ctx.createBufferSource()
    source.buffer = buffer
    source.connect(ctx.destination)
    source.onended = () => {
      if (current === source) current = null
    }
    source.start()
    current = source
  }

  const toggle = () => {
    enabled.value = !enabled.value
    if (!enabled.value && current) {
      try {
        current.stop()
      } catch {
        /* 同上 */
      }
      current = null
    }
  }

  onMounted(load)

  onBeforeUnmount(() => {
    try {
      current?.stop()
    } catch {
      /* 同上 */
    }
    void ctx?.close()
    ctx = null
    buffer = null
  })

  return { enabled, blocked, play, toggle }
}
