import type { Question, GameMode } from '@/types'

/** 上傳照片到房間，回傳 photoId；接著由 WebSocket 送 SUBMIT_PHOTO */
export async function uploadPhoto(roomId: string, blob: Blob): Promise<string> {
  const form = new FormData()
  form.append('photo', blob, 'shot.jpg')

  const res = await fetch(`/api/rooms/${roomId}/photos`, { method: 'POST', body: form })
  if (!res.ok) {
    const detail = await res.json().catch(() => ({ error: '上傳失敗' }))
    throw new Error(detail.error || '上傳失敗')
  }

  const data = await res.json()
  return data.photoId as string
}

/** 取得題庫，給建房頁的題目選擇器 */
export async function fetchQuestionBank(mode: GameMode): Promise<Question[]> {
  const res = await fetch(`/api/questions?mode=${mode}`)
  if (!res.ok) throw new Error('載入題庫失敗')

  const data = await res.json()
  return data.questions as Question[]
}

/** 查房間是否存在，加入頁用來提前提示錯誤房號 */
export async function fetchRoom(roomId: string) {
  const res = await fetch(`/api/rooms/${roomId}`)
  if (!res.ok) return null
  return res.json()
}

// ── 題目示範圖後台 ────────────────────────────────────────

export interface AdminQuestion extends Question {
  hasBuiltin: boolean
  hasCustom: boolean
  imageUrl: string
  disabled: boolean
}

export interface QuestionDraft {
  text: string
  mode: GameMode
  category: string
  difficulty: number
}

const ADMIN_TOKEN_KEY = 'beauty_admin_token'

export const getAdminToken = () => localStorage.getItem(ADMIN_TOKEN_KEY) ?? ''
export const setAdminToken = (token: string) => localStorage.setItem(ADMIN_TOKEN_KEY, token)

function adminHeaders(): HeadersInit {
  const token = getAdminToken()
  return token ? { 'X-Admin-Token': token } : {}
}

async function unwrap(res: Response) {
  if (res.status === 401) throw new Error('後台密碼不正確')
  if (!res.ok) {
    const detail = await res.json().catch(() => ({ error: '操作失敗' }))
    throw new Error(detail.error || '操作失敗')
  }
  return res.json()
}

/** 列出所有題目與示範圖狀態 */
export async function fetchAdminExamples(): Promise<{
  questions: AdminQuestion[]
  storage: string
  protected: boolean
}> {
  return unwrap(await fetch('/api/admin/examples', { headers: adminHeaders() }))
}

/** 上傳一張示範圖蓋掉內建的 */
export async function uploadExample(questionId: number, file: File) {
  const form = new FormData()
  form.append('image', file)

  return unwrap(
    await fetch(`/api/admin/questions/${questionId}/example`, {
      method: 'POST',
      headers: adminHeaders(),
      body: form,
    }),
  )
}

/** 移除自訂圖，還原成內建示範圖 */
export async function deleteExample(questionId: number) {
  return unwrap(
    await fetch(`/api/admin/questions/${questionId}/example`, {
      method: 'DELETE',
      headers: adminHeaders(),
    }),
  )
}

/** 新增一題自訂題目 */
export async function createQuestion(draft: QuestionDraft): Promise<{ question: Question }> {
  return unwrap(
    await fetch('/api/admin/questions', {
      method: 'POST',
      headers: { ...adminHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(draft),
    }),
  )
}

/** 修改自訂題目 */
export async function updateQuestion(
  id: number,
  draft: Omit<QuestionDraft, 'mode'>,
): Promise<{ question: Question }> {
  return unwrap(
    await fetch(`/api/admin/questions/${id}`, {
      method: 'PUT',
      headers: { ...adminHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(draft),
    }),
  )
}

/** 刪除自訂題目（內建題不能刪，只能停用） */
export async function deleteQuestion(id: number) {
  return unwrap(
    await fetch(`/api/admin/questions/${id}`, { method: 'DELETE', headers: adminHeaders() }),
  )
}

/** 停用或重新啟用一題 */
export async function setQuestionDisabled(id: number, disabled: boolean) {
  return unwrap(
    await fetch(`/api/admin/questions/${id}/disabled`, {
      method: 'POST',
      headers: { ...adminHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ disabled }),
    }),
  )
}
