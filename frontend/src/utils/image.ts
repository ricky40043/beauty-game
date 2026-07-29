/** 拍照後在瀏覽器端壓縮，避免把幾 MB 的原圖丟上伺服器 */

const MAX_EDGE = 1080
const QUALITY = 0.75

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

  const blob = await canvasToBlob(canvas)
  return { blob, previewUrl: URL.createObjectURL(blob) }
}

/** 從系統相機或相簿選到的檔案，縮圖後回傳 */
export async function compressFile(file: File): Promise<{ blob: Blob; previewUrl: string }> {
  const bitmap = await loadImage(file)
  const size = fitSize(bitmap.width, bitmap.height)

  const canvas = document.createElement('canvas')
  canvas.width = size.width
  canvas.height = size.height

  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('這台裝置不支援 canvas')
  ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height)

  if ('close' in bitmap && typeof bitmap.close === 'function') bitmap.close()

  const blob = await canvasToBlob(canvas)
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
