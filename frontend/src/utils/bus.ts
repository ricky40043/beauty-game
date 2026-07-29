import type { PhotoSubmission, RoundResult } from '@/types'

/** 一次性事件（照片彈出、得獎公布）用簡單的發布訂閱，不塞進 store 狀態裡 */
type Events = {
  photo: { photo: PhotoSubmission; replaced: boolean }
  roundResult: RoundResult
  roomClosed: { title: string; message: string }
}

type Handler<K extends keyof Events> = (payload: Events[K]) => void

// 內部用寬鬆型別存放，對外的 on / emit 仍是型別安全的
const handlers = new Map<keyof Events, Set<(payload: never) => void>>()

export function on<K extends keyof Events>(event: K, handler: Handler<K>): () => void {
  let set = handlers.get(event)
  if (!set) {
    set = new Set()
    handlers.set(event, set)
  }

  const wrapped = handler as (payload: never) => void
  set.add(wrapped)
  return () => set!.delete(wrapped)
}

export function emit<K extends keyof Events>(event: K, payload: Events[K]): void {
  handlers.get(event)?.forEach((handler) => (handler as Handler<K>)(payload))
}
