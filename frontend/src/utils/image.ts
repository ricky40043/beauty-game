/** 拍照後在瀏覽器端壓縮，避免把幾 MB 的原圖丟上伺服器 */

const MAX_EDGE = 1080
const QUALITY = 0.75

/** 壓縮目標。壓不到就降品質、再不夠就縮尺寸，直到進得來為止。 */
const TARGET_BYTES = 300 * 1024
/** 品質與邊長的下限，避免為了達標把照片壓爛 */
const MIN_QUALITY = 0.4
const MIN_EDGE = 540

/** 把 canvas 內容轉成 JPEG Blob */
function canvasToBlob(canvas: HTMLCanvasElement, quality = QUALITY): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error('壓縮照片失敗'))),
      'image/jpeg',
      quality,
    )
  })
}

/** 依長邊上限算出縮圖尺寸 */
function fitSize(width: number, height: number, maxEdge = MAX_EDGE) {
  const longEdge = Math.max(width, height)
  if (longEdge <= maxEdge) return { width, height }

  const scale = maxEdge / longEdge
  return { width: Math.round(width * scale), height: Math.round(height * scale) }
}

/** 把來源畫到指定尺寸的新 canvas */
function drawTo(source: CanvasImageSource, width: number, height: number): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height

  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('這台裝置不支援 canvas')
  ctx.drawImage(source, 0, 0, width, height)
  return canvas
}

/** 把 toDataURL 的結果轉成 Blob（不用 fetch(data:) 是為了不受 CSP 影響） */
function dataURLToBlob(dataURL: string): Blob {
  const [header, encoded] = dataURL.split(',')
  const binary = atob(encoded)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return new Blob([bytes], { type: header.match(/:(.*?);/)?.[1] || 'image/jpeg' })
}

/**
 * 編碼一次。toBlob 與 toDataURL 在瀏覽器裡是兩套實作，
 * 有些 Safari 版本只有後者會理會 quality，所以要留一條換路的餘地。
 */
async function encode(canvas: HTMLCanvasElement, quality: number, viaDataURL: boolean) {
  return viaDataURL
    ? dataURLToBlob(canvas.toDataURL('image/jpeg', quality))
    : canvasToBlob(canvas, quality)
}

/**
 * 壓到 TARGET_BYTES 以內，而且是**檢查實際輸出大小**、不是呼叫完就當作壓好了。
 *
 * 只設 quality 參數是不夠的：部分 Safari 版本會直接忽略 toBlob 的 quality，
 * 吐回接近無損的 JPEG —— 1080px 的照片因此可以到 1~2MB，而程式完全不會察覺。
 *
 * 對策依序是「降品質 → 換 toDataURL 重來 → 縮尺寸」。換編碼路徑排在縮尺寸前面，
 * 因為它能壓下來又不犧牲解析度；縮尺寸是最後手段。
 */
async function compressToTarget(master: HTMLCanvasElement): Promise<Blob> {
  let canvas = master
  let quality = QUALITY
  let viaDataURL = false
  let blob = await encode(canvas, quality, viaDataURL)

  // 每一輪不是降品質就是縮尺寸；設上限是為了不要在慢手機上卡太久
  for (let attempt = 0; attempt < 8 && blob.size > TARGET_BYTES; attempt++) {
    if (quality > MIN_QUALITY) {
      const before = blob.size
      quality = Math.max(MIN_QUALITY, quality - 0.15)
      blob = await encode(canvas, quality, viaDataURL)

      // 品質降了大小卻幾乎沒動 → 這條路徑沒理會 quality，換 toDataURL 重壓一次
      if (!viaDataURL && blob.size > before * 0.95) {
        viaDataURL = true
        blob = await encode(canvas, quality, viaDataURL)
      }
      continue
    }

    if (Math.max(canvas.width, canvas.height) <= MIN_EDGE) break

    canvas = drawTo(canvas, Math.round(canvas.width * 0.8), Math.round(canvas.height * 0.8))
    blob = await encode(canvas, quality, viaDataURL)
  }

  return blob
}

/** 從 <video> 目前的畫面截一張照片 */
export async function captureFromVideo(
  video: HTMLVideoElement,
  mirrored: boolean,
): Promise<{ blob: Blob; previewUrl: string }> {
  const source = fitSize(video.videoWidth, video.videoHeight)
  const canvas = document.createElement('canvas')
  canvas.width = source.width
  canvas.height = source.height

  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('這台裝置不支援 canvas')

  // 前鏡頭預覽是鏡像的，拍出來也要跟著鏡像才符合使用者看到的畫面
  if (mirrored) {
    ctx.translate(canvas.width, 0)
    ctx.scale(-1, 1)
  }
  ctx.drawImage(video, 0, 0, canvas.width, canvas.height)

  const blob = await compressToTarget(canvas)
  return { blob, previewUrl: URL.createObjectURL(blob) }
}

/** 從系統相機或相簿選到的檔案，縮圖後回傳 */
export async function compressFile(file: File): Promise<{ blob: Blob; previewUrl: string }> {
  const bitmap = await loadImage(file)
  const size = fitSize(bitmap.width, bitmap.height)
  const canvas = drawTo(bitmap, size.width, size.height)

  if ('close' in bitmap && typeof bitmap.close === 'function') bitmap.close()

  const blob = await compressToTarget(canvas)
  return { blob, previewUrl: URL.createObjectURL(blob) }
}

/** createImageBitmap 在舊版 Safari 沒有，退回 <img> 解碼 */
async function loadImage(file: File): Promise<ImageBitmap | HTMLImageElement> {
  if (typeof createImageBitmap === 'function') {
    try {
      return await createImageBitmap(file)
    } catch {
      // 落到下面的 <img> 路徑
    }
  }

  const url = URL.createObjectURL(file)
  try {
    const img = new Image()
    img.src = url
    await img.decode()
    return img
  } finally {
    // decode 完成後 img 已經持有資料，可以安全釋放
    setTimeout(() => URL.revokeObjectURL(url), 0)
  }
}
