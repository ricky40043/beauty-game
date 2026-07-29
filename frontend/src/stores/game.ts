import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type {
  CurrentQuestion,
  GameMode,
  PhotoSubmission,
  Player,
  Question,
  RoomStatus,
  RoundResult,
  ScoreInfo,
} from '@/types'

export const useGameStore = defineStore('game', () => {
  // ── 房間 ────────────────────────────────────────────────
  const roomId = ref('')
  const hostName = ref('')
  const mode = ref<GameMode>('solo')
  const status = ref<RoomStatus>('waiting')
  const requireNickname = ref(false)
  const totalQuestions = ref(0)
  const questionTimeLimit = ref(60)
  const joinUrl = ref('')
  const questions = ref<Question[]>([])

  // ── 我是誰 ──────────────────────────────────────────────
  const playerId = ref('')
  const playerName = ref('')
  const avatar = ref('')
  const isHost = ref(false)
  const hostToken = ref('')

  // ── 進行中的一題 ────────────────────────────────────────
  const players = ref<Player[]>([])
  const question = ref<CurrentQuestion | null>(null)
  const timeLeft = ref(0)
  const roundPhotos = ref<PhotoSubmission[]>([])
  const myPhotoUrl = ref('')
  const uploading = ref(false)

  // ── 結果 ────────────────────────────────────────────────
  const scores = ref<ScoreInfo[]>([])
  const roundResult = ref<RoundResult | null>(null)
  const history = ref<RoundResult[]>([])

  // 前 5 張（依抵達順序），主畫面常駐縮圖列用
  const topFive = computed(() => roundPhotos.value.slice(0, 5))

  const submittedCount = computed(() => roundPhotos.value.length)
  const connectedCount = computed(() => players.value.filter((p) => p.isConnected).length)
  const hasSubmitted = computed(() =>
    roundPhotos.value.some((p) => p.playerId === playerId.value),
  )

  const setRoom = (payload: Record<string, any>) => {
    if (payload.roomId) roomId.value = payload.roomId
    if (payload.hostName !== undefined) hostName.value = payload.hostName
    if (payload.mode) mode.value = payload.mode
    if (payload.status) status.value = payload.status
    if (payload.requireNickname !== undefined) requireNickname.value = payload.requireNickname
    if (payload.totalQuestions !== undefined) totalQuestions.value = payload.totalQuestions
    if (payload.questionTimeLimit !== undefined) questionTimeLimit.value = payload.questionTimeLimit
    if (payload.joinUrl) joinUrl.value = payload.joinUrl
    if (Array.isArray(payload.questions)) questions.value = payload.questions
    if (Array.isArray(payload.players)) players.value = payload.players
  }

  const setMe = (payload: {
    playerId?: string
    playerName?: string
    avatar?: string
    isHost?: boolean
    hostToken?: string
  }) => {
    if (payload.playerId) playerId.value = payload.playerId
    if (payload.playerName) playerName.value = payload.playerName
    if (payload.avatar) avatar.value = payload.avatar
    if (payload.isHost !== undefined) isHost.value = payload.isHost
    if (payload.hostToken) hostToken.value = payload.hostToken
  }

  const setPlayers = (list: Player[]) => {
    players.value = list ?? []
  }

  /** 新的一題：清掉上一題的照片與結果 */
  const startQuestion = (payload: CurrentQuestion) => {
    question.value = payload
    timeLeft.value = payload.timeLimit
    totalQuestions.value = payload.totalQuestions
    roundPhotos.value = []
    myPhotoUrl.value = ''
    roundResult.value = null
    status.value = 'shooting'
  }

  /** 收到一張投稿。覆蓋（重拍）時原地換掉，不新增一筆 */
  const applyPhoto = (photo: PhotoSubmission, replaced: boolean) => {
    if (replaced) {
      const index = roundPhotos.value.findIndex((p) => p.playerId === photo.playerId)
      if (index >= 0) {
        roundPhotos.value.splice(index, 1, photo)
        if (photo.playerId === playerId.value) myPhotoUrl.value = photo.url
        return
      }
    }

    roundPhotos.value.push(photo)
    if (photo.playerId === playerId.value) myPhotoUrl.value = photo.url
  }

  const closeRound = (photos: PhotoSubmission[]) => {
    if (Array.isArray(photos)) roundPhotos.value = photos
    timeLeft.value = 0
    status.value = 'judging'
  }

  const applyRoundResult = (result: RoundResult, nextScores: ScoreInfo[]) => {
    roundResult.value = result
    if (Array.isArray(nextScores)) scores.value = nextScores
    status.value = 'round_result'
  }

  const finish = (finalScores: ScoreInfo[], finalHistory: RoundResult[]) => {
    if (Array.isArray(finalScores)) scores.value = finalScores
    if (Array.isArray(finalHistory)) history.value = finalHistory
    status.value = 'finished'
    question.value = null
    timeLeft.value = 0
  }

  const resetToLobby = () => {
    status.value = 'waiting'
    question.value = null
    timeLeft.value = 0
    roundPhotos.value = []
    myPhotoUrl.value = ''
    roundResult.value = null
    history.value = []
    scores.value = []
  }

  const reset = () => {
    roomId.value = ''
    hostName.value = ''
    mode.value = 'solo'
    status.value = 'waiting'
    requireNickname.value = false
    totalQuestions.value = 0
    joinUrl.value = ''
    questions.value = []
    playerId.value = ''
    playerName.value = ''
    avatar.value = ''
    isHost.value = false
    hostToken.value = ''
    players.value = []
    resetToLobby()
  }

  return {
    roomId,
    hostName,
    mode,
    status,
    requireNickname,
    totalQuestions,
    questionTimeLimit,
    joinUrl,
    questions,
    playerId,
    playerName,
    avatar,
    isHost,
    hostToken,
    players,
    question,
    timeLeft,
    roundPhotos,
    myPhotoUrl,
    uploading,
    scores,
    roundResult,
    history,
    topFive,
    submittedCount,
    connectedCount,
    hasSubmitted,
    setRoom,
    setMe,
    setPlayers,
    startQuestion,
    applyPhoto,
    closeRound,
    applyRoundResult,
    finish,
    resetToLobby,
    reset,
  }
})
