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
  /** 試玩題目，空字串 = 用該模式的預設題（PRACTICE_PRESETS 的第一個） */
  practiceText: string
  /** 開始遊戲前主畫面先播玩法說明 */
  gameIntro: boolean
}

/** 試玩題的現成選項，第一個是預設（要跟後端 DefaultPracticeText 一致） */
export const PRACTICE_PRESETS: Record<GameMode, string[]> = {
  solo: ['比出小愛心', '踩到狗大便時的表情'],
  group: ['全員一起比 YA'],
}

/** 主畫面正在播玩法說明時，轉播給玩家手機的頁次 */
export interface IntroState {
  page: number
  total: number
  title: string
}

/** 單人隨機抽題可以勾的分類（團體題只有一類，不用選） */
export const SOLO_CATEGORIES = [
  'expression',
  'imitate',
  'pose',
  'color',
  'object',
  'student',
  'work',
  'church',
  'christmas',
]

/** 主畫面題目上方的分類小標籤用 */
export const CATEGORY_EMOJI: Record<string, string> = {
  expression: '😆',
  imitate: '🎭',
  pose: '🤳',
  color: '🎨',
  object: '🔍',
  advanced: '⭐',
  student: '🎒',
  work: '💼',
  church: '⛪',
  christmas: '🎄',
  group: '👯',
  custom: '✏️',
}

export const CATEGORY_LABELS: Record<string, string> = {
  expression: '表情',
  imitate: '模仿',
  pose: '自拍姿勢',
  color: '找顏色',
  object: '找物品',
  advanced: '進階',
  student: '學生篇',
  work: '職場篇',
  church: '教會',
  christmas: '聖誕市集',
  group: '團體',
  custom: '自訂',
}
