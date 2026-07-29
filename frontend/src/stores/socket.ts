import { defineStore } from 'pinia'
import { ref } from 'vue'
import router from '@/router'
import { useGameStore } from './game'
import { useUIStore } from './ui'
import { emit } from '@/utils/bus'
import type { RoomSetup, StoredSession, WebSocketMessage } from '@/types'

const SESSION_KEY = 'beauty_game_session'
const MAX_RECONNECT = 5

function resolveWsUrl() {
  const fromEnv = import.meta.env.VITE_WS_URL?.toString().trim()
  if (fromEnv) return fromEnv

  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.host}/ws`
}

export function loadSession(): StoredSession | null {
  const raw = localStorage.getItem(SESSION_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as StoredSession
  } catch {
    localStorage.removeItem(SESSION_KEY)
    return null
  }
}

function saveSession(session: StoredSession) {
  localStorage.setItem(SESSION_KEY, JSON.stringify(session))
}

export function clearSession() {
  localStorage.removeItem(SESSION_KEY)
}

export const useSocketStore = defineStore('socket', () => {
  const socket = ref<WebSocket | null>(null)
  const isConnected = ref(false)
  const reconnectAttempts = ref(0)
  const shouldReconnect = ref(true)

  const blockingDialog = ref<{ title: string; message: string } | null>(null)

  const game = useGameStore()
  const ui = useUIStore()

  // 伺服器常常連續送兩則訊息指向同一個畫面（例如 GAME_STARTED 後面立刻跟著
  // NEW_QUESTION）。router.push 是非同步的，在它完成前 currentRoute 還是舊路徑，
  // 直接比對會漏判而發出重複導航 —— 所以額外記住「正在前往哪裡」。
  let pendingPath = ''

  const navigate = (path: string) => {
    if (path === pendingPath || path === router.currentRoute.value.path) return

    pendingPath = path
    router
      .push(path)
      // 導航被後續操作取代時 vue-router 會 reject，這是預期行為，不該變成未捕捉的錯誤
      .catch(() => {})
      .finally(() => {
        if (pendingPath === path) pendingPath = ''
      })
  }

  const openBlockingDialog = (title: string, message: string) => {
    if (blockingDialog.value) return
    shouldReconnect.value = false
    blockingDialog.value = { title, message }
  }

  const acknowledgeDialog = () => {
    blockingDialog.value = null
    clearSession()
    game.reset()
    disconnect()
    shouldReconnect.value = true
    navigate('/')
  }

  const connect = () => {
    shouldReconnect.value = true

    if (socket.value && socket.value.readyState <= WebSocket.OPEN) return

    const ws = new WebSocket(resolveWsUrl())
    socket.value = ws

    ws.onopen = () => {
      isConnected.value = true
      reconnectAttempts.value = 0

      // 有舊 session 就先問伺服器房間還在不在，再決定要不要重連
      const session = loadSession()
      if (session?.roomId) {
        send('CHECK_ROOM', { roomId: session.roomId, playerId: session.playerId })
      }
    }

    ws.onclose = () => {
      isConnected.value = false
      if (!shouldReconnect.value) return

      if (reconnectAttempts.value >= MAX_RECONNECT) {
        openBlockingDialog(
          '連線已中斷',
          '和伺服器的連線中斷且無法自動恢復，點「確認」回到首頁重新開始。',
        )
        return
      }

      reconnectAttempts.value += 1
      const delay = Math.min(1500 * 1.6 ** (reconnectAttempts.value - 1), 8000)
      ui.showWarning(`連線中斷，${Math.round(delay / 1000)} 秒後重試…`)
      setTimeout(() => {
        if (!isConnected.value && shouldReconnect.value) connect()
      }, delay)
    }

    ws.onmessage = (event) => {
      try {
        handleMessage(JSON.parse(event.data) as WebSocketMessage)
      } catch {
        // 收到非 JSON 內容就略過
      }
    }
  }

  const disconnect = () => {
    shouldReconnect.value = false
    socket.value?.close()
    socket.value = null
    isConnected.value = false
  }

  const send = (type: string, data: Record<string, unknown> = {}) => {
    if (socket.value?.readyState === WebSocket.OPEN) {
      socket.value.send(JSON.stringify({ type, data }))
      return
    }
    ui.showError('連線還沒建立，請稍等一下再試')
  }

  // ── 訊息處理 ────────────────────────────────────────────

  function handleMessage(msg: WebSocketMessage) {
    const data = (msg.data ?? {}) as Record<string, any>

    switch (msg.type) {
      case 'ROOM_CREATED':
        game.setRoom(data)
        game.setMe({ playerId: data.clientId, isHost: true, hostToken: data.hostToken })
        saveSession({
          roomId: data.roomId,
          playerId: data.clientId,
          playerName: data.hostName,
          isHost: true,
          hostToken: data.hostToken,
        })
        navigate(`/lobby/${data.roomId}`)
        break

      case 'JOIN_SUCCESS':
        game.setRoom(data)
        game.setMe({
          playerId: data.playerId,
          playerName: data.playerName,
          avatar: data.avatar,
          isHost: false,
        })
        saveSession({
          roomId: data.roomId,
          playerId: data.playerId,
          playerName: data.playerName,
          isHost: false,
        })
        ui.showSuccess(`歡迎，${data.playerName}！`)
        // 中途加入時遊戲可能已經在跑，直接進對應畫面，不要先閃一下大廳
        routeByStatus(data.roomId, data.status)
        break

      case 'PLAYER_JOINED':
      case 'PLAYER_LEFT':
      case 'PLAYER_DISCONNECTED':
      case 'PLAYER_REJOINED':
      case 'PLAYER_RENAMED':
        game.setPlayers(data.players ?? [])
        break

      case 'RENAME_SUCCESS':
        game.setMe({ playerName: data.playerName })
        ui.showSuccess('暱稱已更新')
        break

      case 'SETTINGS_UPDATED':
        game.setRoom(data)
        ui.showInfo('房間設定已更新')
        break

      case 'ROOM_STATUS':
        handleRoomStatus(data)
        break

      case 'REJOIN_SUCCESS':
        handleRejoinSuccess(data)
        break

      case 'GAME_STARTED':
        game.setRoom(data)
        goToGameView(data.roomId)
        break

      case 'NEW_QUESTION':
        game.startQuestion({
          questionId: data.questionId,
          text: data.text,
          category: data.category,
          difficulty: data.difficulty,
          questionIndex: data.questionIndex,
          questionNum: data.questionNum,
          totalQuestions: data.totalQuestions,
          timeLimit: data.timeLimit,
          isPractice: Boolean(data.isPractice),
        })
        goToGameView(data.roomId)
        break

      case 'TIMER_UPDATE':
        game.timeLeft = data.timeLeft
        break

      case 'PHOTO_SUBMITTED':
        game.applyPhoto(data.photo, Boolean(data.replaced))
        emit('photo', { photo: data.photo, replaced: Boolean(data.replaced) })
        break

      case 'PHOTO_ACCEPTED':
        game.uploading = false
        game.myPhotoUrl = data.url
        ui.showSuccess(data.replaced ? '已換成新的照片！' : '上傳成功！')
        break

      case 'PRACTICE_ENDED':
        ui.showSuccess('試玩結束，正式開始！')
        break

      case 'ROUND_CLOSED':
        game.closeRound(data.photos ?? [])
        break

      case 'ROUND_RESULT':
        game.applyRoundResult(
          {
            questionIndex: data.questionIndex,
            questionText: data.questionText,
            totalPhotos: data.totalPhotos ?? 0,
            winners: data.winners ?? [],
            groupBonus: data.groupBonus ?? 0,
          },
          data.scores ?? [],
        )
        emit('roundResult', {
          questionIndex: data.questionIndex,
          questionText: data.questionText,
          totalPhotos: data.totalPhotos ?? 0,
          winners: data.winners ?? [],
          groupBonus: data.groupBonus ?? 0,
        })
        break

      case 'GAME_FINISHED':
        game.finish(data.scores ?? [], data.history ?? [])
        navigate(`/results/${data.roomId}`)
        break

      case 'ROOM_RESET_TO_LOBBY':
        game.resetToLobby()
        game.setPlayers(data.players ?? [])
        ui.showSuccess('房主開了新的一局！')
        navigate(`/lobby/${data.roomId}`)
        break

      case 'HOST_DISCONNECTED':
        ui.showWarning('主畫面斷線了，正在等它回來…')
        break

      case 'HOST_RECONNECTED':
        ui.showSuccess('主畫面已恢復連線')
        break

      case 'ROOM_CLOSED':
        openBlockingDialog('房間已關閉', data.reason || '這個房間已經結束了。')
        break

      case 'ERROR':
        handleError(data)
        break

      case 'PONG':
        break
    }
  }

  /** 目前這頁是不是綁在某個房間上（大廳、遊戲中、結算） */
  const onRoomPage = () => /^\/(lobby|game|results)\//.test(router.currentRoute.value.path)

  function handleRoomStatus(data: Record<string, any>) {
    const session = loadSession()
    if (!session) return

    if (!data.exists) {
      clearSession()
      game.reset()
      // 房間沒了卻放著使用者停在遊戲頁，畫面會變成沒有任何按鈕的死路。
      // 最常見的原因是伺服器重新部署過 —— 房間只存在記憶體，重啟就全沒了。
      if (onRoomPage()) {
        openBlockingDialog(
          '房間已經不存在',
          '這個房間找不到了。伺服器重新啟動（例如剛部署新版）會清掉所有進行中的房間，請重新開一間。',
        )
      }
      return
    }

    if (!data.playerExists && !session.hostToken) {
      clearSession()
      game.reset()
      if (onRoomPage()) {
        openBlockingDialog('已離開房間', '你已經不在這個房間裡了，請重新加入。')
      }
      return
    }

    send('REJOIN_ROOM', {
      roomId: session.roomId,
      playerId: session.playerId,
      hostToken: session.hostToken ?? '',
    })
  }

  function handleRejoinSuccess(data: Record<string, any>) {
    game.setRoom(data)
    game.setMe({
      playerId: data.isHost ? data.clientId : data.playerId,
      playerName: data.playerName,
      avatar: data.avatar,
      isHost: Boolean(data.isHost),
      hostToken: data.hostToken,
    })

    game.setPlayers(data.players ?? [])
    game.scores = data.scores ?? []
    game.history = data.history ?? []
    game.roundPhotos = data.roundPhotos ?? []
    game.timeLeft = data.timeLeft ?? 0

    if (data.question) {
      game.question = {
        questionId: data.question.id,
        text: data.question.text,
        category: data.question.category,
        difficulty: data.question.difficulty,
        questionIndex: data.currentQuestion,
        questionNum: data.currentQuestion + 1,
        totalQuestions: data.totalQuestions,
        timeLimit: data.questionTimeLimit,
      }
    }

    const mine = (data.roundPhotos ?? []).find(
      (p: any) => p.playerId === (data.playerId ?? data.clientId),
    )
    game.myPhotoUrl = mine?.url ?? ''

    const session = loadSession()
    saveSession({
      roomId: data.roomId,
      playerId: data.isHost ? data.clientId : data.playerId,
      playerName: data.playerName ?? session?.playerName ?? '',
      isHost: Boolean(data.isHost),
      hostToken: data.hostToken ?? session?.hostToken,
    })

    ui.showSuccess('已恢復連線')
    routeByStatus(data.roomId, data.status)
  }

  function handleError(data: Record<string, any>) {
    ui.showError(data.message || '發生錯誤')

    // 只有在我們手上根本沒有房間時才回首頁。
    // 房間真的被關掉會另外收到 ROOM_CLOSED，那條才是權威來源；
    // 這裡若無條件跳首頁，一則過時的錯誤就會把正在玩的人踢出去。
    if ((data.code === 'ROOM_NOT_FOUND' || data.code === 'PLAYER_NOT_FOUND') && !game.roomId) {
      clearSession()
      game.reset()
      navigate('/')
    }
    if (data.code === 'NOT_SHOOTING' || data.code === 'UPLOAD_LIMIT') {
      game.uploading = false
    }
  }

  function goToGameView(id: string) {
    navigate(game.isHost ? `/game/host/${id}` : `/game/player/${id}`)
  }

  function routeByStatus(id: string, status: string) {
    if (status === 'finished') {
      navigate(`/results/${id}`)
      return
    }
    if (status === 'waiting') {
      navigate(`/lobby/${id}`)
      return
    }
    goToGameView(id)
  }

  // ── 對外動作 ────────────────────────────────────────────

  const createRoom = (setup: RoomSetup) => {
    clearSession()
    game.reset()
    send('CREATE_ROOM', {
      hostName: setup.hostName,
      mode: setup.mode,
      totalQuestions: setup.totalQuestions,
      questionTimeLimit: setup.questionTimeLimit,
      difficulty: setup.difficulty,
      questionMode: setup.questionMode,
      questionIds: setup.questionIds,
      customQuestions: setup.customQuestions,
      requireNickname: setup.requireNickname,
      practiceRound: setup.practiceRound,
    })
  }

  const joinRoom = (roomId: string, playerName = '') => {
    clearSession()
    game.reset()
    send('JOIN_ROOM', { roomId: roomId.toUpperCase(), playerName })
  }

  const rename = (name: string) => send('RENAME_PLAYER', { name })
  const updateSettings = (payload: Record<string, unknown>) => send('UPDATE_SETTINGS', payload)
  const startGame = () => send('START_GAME')
  const submitPhoto = (photoId: string) => send('SUBMIT_PHOTO', { photoId })
  const endShooting = () => send('END_SHOOTING')
  const endPractice = () => send('END_PRACTICE')
  const pickWinners = (photoIds: string[]) => send('PICK_WINNERS', { photoIds })
  const skipRound = () => send('SKIP_ROUND')
  const nextQuestion = () => send('NEXT_QUESTION')
  const resetRoomToLobby = () => send('RESET_ROOM_TO_LOBBY')

  const leaveRoom = () => {
    send('LEAVE_ROOM')
    clearSession()
    game.reset()
    navigate('/')
  }

  return {
    isConnected,
    blockingDialog,
    connect,
    disconnect,
    acknowledgeDialog,
    createRoom,
    joinRoom,
    rename,
    updateSettings,
    startGame,
    submitPhoto,
    endShooting,
    endPractice,
    pickWinners,
    skipRound,
    nextQuestion,
    resetRoomToLobby,
    leaveRoom,
  }
})
