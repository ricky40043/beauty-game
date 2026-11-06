export type GameMode = 'solo' | 'group'

export type RoomStatus =
  | 'waiting'
  | 'shooting'
  | 'judging'
  | 'voting'
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
  /** 投票模式拿到的票數 */
  votes?: number
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
  isPractice?: boolean
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
  /** 隨機模式只從這些分類平均抽，空陣列代表全部 */
  categories: string[]
  /** 每題得獎名額，0 = 全部投稿都能排名 */
  maxWinners: number
  /** 得獎畫面停幾秒自動下一題，0 = 手動 */
  resultSeconds: number
  /** 投票模式：玩家投票決定名次 */
  voteMode: boolean
  /** 投票模式的候選張數（最快的前 N 張） */
  voteCandidates: number
  /** 投票倒數秒數 */
  voteSeconds: number
  requireNickname: boolean
  practiceRound: boolean
}

/** 單人隨機抽題可以勾的分類（團體題只有一類，不用選） */
export const SOLO_CATEGORIES = [
  'expression',
  'imitate',
  'pose',
  'color',
  'object',
  'advanced',
  'student',
  'work',
  'church',
]

export const CATEGORY_LABELS: Record<string, string> = {
  expression: '表情',
  imitate: '模仿',
  pose: '自拍姿勢',
  color: '找顏色',
  object: '找物品',
  advanced: '進階',
  student: '學生篇',
  work: '職場篇',
  church: '教會聖誕',
  group: '團體',
  custom: '自訂',
}
