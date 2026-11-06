import { onBeforeUnmount } from 'vue'

const STORAGE_KEY = 'beauty_sound_enabled'

/**
 * 冠軍揭曉的號角聲與掌聲。
 *
 * 全部用 Web Audio 即時合成，不打包音檔：歡呼、掌聲這類素材很難找到授權乾淨的版本，
 * 而 App 會對外部署。跟快門聲共用同一個開關（主畫面右上角的喇叭）。
 */
export function useCheer() {
  let ctx: AudioContext | null = null

  const enabled = () => {
    try {
      return localStorage.getItem(STORAGE_KEY) !== 'false'
    } catch {
      return true
    }
  }

  const audio = () => {
    if (!enabled()) return null
    if (!ctx) {
      const AudioCtx = window.AudioContext ?? (window as any).webkitAudioContext
      if (!AudioCtx) return null
      ctx = new AudioCtx() as AudioContext
    }
    // 房主一路按鈕點進結算頁，頁面已經有互動過，resume 不會被擋
    if (ctx.state === 'suspended') void ctx.resume()
    return ctx
  }

  /** 揭曉瞬間的低頻重擊，讓畫面「碰」一下有份量 */
  const boom = () => {
    const ac = audio()
    if (!ac) return
    const t = ac.currentTime
    const osc = ac.createOscillator()
    const gain = ac.createGain()
    osc.type = 'sine'
    osc.frequency.setValueAtTime(110, t)
    osc.frequency.exponentialRampToValueAtTime(38, t + 0.6)
    gain.gain.setValueAtTime(0.9, t)
    gain.gain.exponentialRampToValueAtTime(0.001, t + 0.7)
    osc.connect(gain).connect(ac.destination)
    osc.start(t)
    osc.stop(t + 0.75)
  }

  /** Do-Mi-Sol 三連音接一個長和弦 */
  const fanfare = () => {
    const ac = audio()
    if (!ac) return
    const t0 = ac.currentTime + 0.05

    const master = ac.createGain()
    master.gain.value = 0.22
    const filter = ac.createBiquadFilter()
    filter.type = 'lowpass'
    filter.frequency.value = 2600
    master.connect(filter).connect(ac.destination)

    const note = (freq: number, start: number, length: number) => {
      // 鋸齒波 + 微微走音的方波，聽起來比較像銅管而不是電子嗶聲
      for (const [type, detune] of [
        ['sawtooth', 0],
        ['square', 7],
      ] as const) {
        const osc = ac.createOscillator()
        const gain = ac.createGain()
        osc.type = type
        osc.frequency.value = freq
        osc.detune.value = detune
        gain.gain.setValueAtTime(0, start)
        gain.gain.linearRampToValueAtTime(0.5, start + 0.02)
        gain.gain.setValueAtTime(0.5, start + length - 0.08)
        gain.gain.linearRampToValueAtTime(0, start + length)
        osc.connect(gain).connect(master)
        osc.start(start)
        osc.stop(start + length + 0.05)
      }
    }

    note(523.25, t0, 0.13)
    note(659.25, t0 + 0.13, 0.13)
    note(783.99, t0 + 0.26, 0.13)
    for (const freq of [523.25, 659.25, 783.99, 1046.5]) note(freq, t0 + 0.42, 1.3)
  }

  /** 一群人拍手：一段預先算好的雜訊，裡面撒了幾百個短促的「啪」 */
  const applause = (seconds = 3.2) => {
    const ac = audio()
    if (!ac) return

    const rate = ac.sampleRate
    const length = Math.floor(rate * seconds)
    const buffer = ac.createBuffer(2, length, rate)
    const clapLength = Math.floor(rate * 0.025)
    const claps = Math.floor(seconds * 140)

    for (let ch = 0; ch < 2; ch++) {
      const data = buffer.getChannelData(ch)
      for (let c = 0; c < claps; c++) {
        const at = Math.floor(Math.random() * (length - clapLength))
        const loud = 0.15 + Math.random() * 0.25
        for (let i = 0; i < clapLength; i++) {
          data[at + i] += (Math.random() * 2 - 1) * loud * Math.exp(-i / (clapLength / 5))
        }
      }
    }

    const source = ac.createBufferSource()
    source.buffer = buffer
    const band = ac.createBiquadFilter()
    band.type = 'bandpass'
    band.frequency.value = 1400
    band.Q.value = 0.6
    const gain = ac.createGain()
    const t = ac.currentTime
    gain.gain.setValueAtTime(0, t)
    gain.gain.linearRampToValueAtTime(0.9, t + 0.35)
    gain.gain.setValueAtTime(0.9, t + seconds - 1.4)
    gain.gain.linearRampToValueAtTime(0, t + seconds)
    source.connect(band).connect(gain).connect(ac.destination)
    source.start(t)
  }

  onBeforeUnmount(() => {
    void ctx?.close()
    ctx = null
  })

  return { boom, fanfare, applause }
}
