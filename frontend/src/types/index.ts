export type GameMode = 'solo' | 'group'

export type RoomStatus =
  | 'waiting'
  | 'shooting'
  | 'judging'
  | 'round_result'
  | 'finished'
  | 'abandoned'

export interface Question {
  id: number
  text: string
  category: string
  difficulty: number
  mode: GameMode
  isCustom: boolean
}

export interface Player {
  id: string
  name: string
  avatar: string
  roomId: string
  score: number
  wins: number
  uploads: number
  isConnected: boolean
}

export interface PhotoSubmission {
  photoId: string
  url: string
  roomId: string
  questionIndex: number
  playerId: string
  playerName: string
  playerAvatar: string
  order: number
  elapsedSec: number
  submittedAt: string
}

export interface RoundWinner {
  photoId: string
  url: string
  playerId: string
  playerName: string
  playerAvatar: string
  rank: number
  points: number
}

export interface RoundResult {
  questionIndex: number
  questionText: string
  totalPhotos: number
  winners: RoundWinner[]
  groupBonus: number
}

export interface ScoreInfo {
  playerId: string
  playerName: string
  avatar: string
  score: number
  rank: number
  wins: number
  uploads: number
  gained: number
}

export interface CurrentQuestion {
  questionId: number
  text: string
  category: string
  difficulty: number
  questionIndex: number
  questionNum: number
  totalQuestions: number
  timeLimit: number
}

export interface WebSocketMessage {
  type: string
  data: Record<string, unknown>
}

export interface StoredSession {
  roomId: string
  playerId: string
  playerName: string
  isHost: boolean
  hostToken?: string
}

/** 建房設定，CreateRoomView 與大廳共用 */
export interface RoomSetup {
  hostName: string
  mode: GameMode
  questionMode: 'random' | 'custom'
  totalQuestions: number
  questionTimeLimit: number
  difficulty: 'basic' | 'mixed'
  questionIds: number[]
  customQuestions: string[]
  requireNickname: boolean
}

export const CATEGORY_LABELS: Record<string, string> = {
  expression: '表情',
  imitate: '模仿',
  pose: '自拍姿勢',
  color: '找顏色',
  object: '找物品',
  advanced: '進階',
  group: '團體',
  custom: '自訂',
}
