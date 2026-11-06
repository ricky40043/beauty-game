/** 一張要放進拼貼圖的得獎照片 */
export interface CollageItem {
  url: string
  questionNum: number
  questionText: string
  playerName: string
  playerAvatar: string
}

const CELL = 480
const GAP = 28
const PAD = 56
const CAPTION = 104
const HEADER = 190
const FONT = '"PingFang TC", "Noto Sans TC", "Microsoft JhengHei", system-ui, sans-serif'

const loadImage = (url: string) =>
  new Promise<HTMLImageElement | null>((resolve) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => resolve(null)
    img.src = url
  })

/** 超出寬度就截斷加「…」 */
const fitText = (ctx: CanvasRenderingContext2D, text: string, maxWidth: number) => {
  if (ctx.measureText(text).width <= maxWidth) return text
  let out = text
  while (out.length > 1 && ctx.measureText(out + '…').width > maxWidth) out = out.slice(0, -1)
  return out + '…'
}

const roundRect = (ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) => {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
}

/**
 * 把每題冠軍畫成一張大圖。全部在瀏覽器裡用 canvas 做：照片在結算頁已經載過、
 * 同源不會污染 canvas，十幾張大概幾百毫秒就好，不用讓伺服器處理圖片。
 */
export async function buildCollage(items: CollageItem[], title: string, subtitle: string): Promise<Blob> {
  const n = items.length
  const cols = n <= 2 ? n : n <= 4 ? 2 : n <= 9 ? 3 : n <= 16 ? 4 : 5
  const rows = Math.ceil(n / cols)
  const width = PAD * 2 + cols * CELL + (cols - 1) * GAP
  const height = HEADER + PAD + rows * (CELL + CAPTION) + (rows - 1) * GAP

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')!

  const bg = ctx.createLinearGradient(0, 0, width, height)
  bg.addColorStop(0, '#2a0f24')
  bg.addColorStop(0.5, '#0b0a1f')
  bg.addColorStop(1, '#05060f')
  ctx.fillStyle = bg
  ctx.fillRect(0, 0, width, height)

  ctx.textAlign = 'center'
  ctx.fillStyle = '#ffd166'
  ctx.font = `900 72px ${FONT}`
  ctx.fillText(title, width / 2, 110)
  ctx.fillStyle = 'rgba(255,255,255,0.6)'
  ctx.font = `500 30px ${FONT}`
  ctx.fillText(subtitle, width / 2, 158)

  const images = await Promise.all(items.map((item) => loadImage(item.url)))

  items.forEach((item, i) => {
    const col = i % cols
    const row = Math.floor(i / cols)
    // 最後一排置中，不要整排靠左留一塊空
    const inRow = row === rows - 1 ? n - row * cols : cols
    const offset = ((cols - inRow) * (CELL + GAP)) / 2
    const x = PAD + offset + col * (CELL + GAP)
    const y = HEADER + row * (CELL + CAPTION + GAP)

    ctx.fillStyle = '#ffffff'
    roundRect(ctx, x - 8, y - 8, CELL + 16, CELL + CAPTION + 8, 22)
    ctx.fill()

    const img = images[i]
    ctx.save()
    roundRect(ctx, x, y, CELL, CELL, 14)
    ctx.clip()
    if (img) {
      // 置中裁成正方形
      const side = Math.min(img.naturalWidth, img.naturalHeight)
      const sx = (img.naturalWidth - side) / 2
      const sy = (img.naturalHeight - side) / 2
      ctx.drawImage(img, sx, sy, side, side, x, y, CELL, CELL)
    } else {
      ctx.fillStyle = '#1e293b'
      ctx.fillRect(x, y, CELL, CELL)
    }
    ctx.restore()

    ctx.textAlign = 'left'
    ctx.fillStyle = '#f92e83'
    ctx.font = `800 26px ${FONT}`
    const tag = `第 ${item.questionNum} 題`
    ctx.fillText(tag, x + 6, y + CELL + 40)
    const tagWidth = ctx.measureText(tag).width + 14
    ctx.fillStyle = '#334155'
    ctx.font = `600 24px ${FONT}`
    ctx.fillText(fitText(ctx, item.questionText, CELL - tagWidth - 8), x + 6 + tagWidth, y + CELL + 40)

    ctx.fillStyle = '#0f172a'
    ctx.font = `800 30px ${FONT}`
    ctx.fillText(fitText(ctx, `🏆 ${item.playerAvatar} ${item.playerName}`, CELL - 12), x + 6, y + CELL + 84)
  })

  return new Promise((resolve, reject) =>
    canvas.toBlob((blob) => (blob ? resolve(blob) : reject(new Error('產生圖片失敗'))), 'image/jpeg', 0.9),
  )
}

/** 把 blob 存成檔案 */
export function saveBlob(blob: Blob, fileName: string) {
  const href = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = href
  a.download = fileName
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(href), 10_000)
}
