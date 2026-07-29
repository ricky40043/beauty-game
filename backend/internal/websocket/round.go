package websocket

import (
	"log"
	"time"

	"beauty-game/internal/models"
	"beauty-game/internal/services"
)

// questionPayload 組出 NEW_QUESTION 的內容
func (h *Hub) questionPayload(room *models.Room, q models.Question) map[string]any {
	return map[string]any{
		"roomId":         room.ID,
		"questionId":     q.ID,
		"text":           q.Text,
		"category":       q.Category,
		"difficulty":     q.Difficulty,
		"mode":           room.Mode,
		"questionIndex":  room.CurrentQuestion,
		"questionNum":    room.CurrentQuestion + 1,
		"totalQuestions": len(room.Questions),
		"timeLimit":      room.QuestionTimeLimit,
		"timeLeft":       room.TimeLeft,
	}
}

// startRound 廣播新題目並啟動倒數
func (h *Hub) startRound(room *models.Room) {
	q, ok := room.CurrentQuestionData()
	if !ok {
		h.finishGame(room)
		return
	}

	log.Printf("📸 房間 %s 第 %d 題：%s", room.ID, room.CurrentQuestion+1, q.Text)
	h.BroadcastToRoom(room.ID, "NEW_QUESTION", h.questionPayload(room, q))

	go h.runRoundTimer(room, room.RoundSeq)
}

// runRoundTimer 每秒推一次倒數；房間換題後 seq 對不上就自動結束
func (h *Hub) runRoundTimer(room *models.Room, seq int) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		timeLeft, expired, stale := h.gameService.Tick(room, seq)
		if stale {
			return
		}

		h.BroadcastToRoom(room.ID, "TIMER_UPDATE", map[string]any{
			"roomId":        room.ID,
			"questionIndex": room.CurrentQuestion,
			"timeLeft":      timeLeft,
		})

		if expired {
			h.closeRound(room, "時間到！")
			return
		}
	}
}

// closeRound 收桌並進入評選階段
func (h *Hub) closeRound(room *models.Room, reason string) {
	if room.Status != models.RoomStatusShooting {
		return
	}

	photos := h.gameService.CloseRound(room)

	h.BroadcastToRoom(room.ID, "ROUND_CLOSED", map[string]any{
		"roomId":        room.ID,
		"questionIndex": room.CurrentQuestion,
		"reason":        reason,
		"photos":        photos,
		"totalPhotos":   len(photos),
	})

	// 一張都沒有就沒得選，直接讓房主按下一題
	if len(photos) == 0 {
		result := h.gameService.SkipRound(room)
		h.BroadcastToRoom(room.ID, "ROUND_RESULT", map[string]any{
			"roomId":        room.ID,
			"questionIndex": result.QuestionIndex,
			"questionText":  result.QuestionText,
			"winners":       result.Winners,
			"groupBonus":    0,
			"scores":        h.gameService.Leaderboard(room),
			"empty":         true,
		})
	}
}

// finishGame 結束整場並送出最終排行榜
func (h *Hub) finishGame(room *models.Room) {
	scores := h.gameService.FinishGame(room)

	h.BroadcastToRoom(room.ID, "GAME_FINISHED", map[string]any{
		"roomId":         room.ID,
		"scores":         scores,
		"history":        room.History,
		"totalQuestions": len(room.Questions),
	})

	log.Printf("🏁 房間 %s 遊戲結束", room.ID)
}

// ─── 房主操作 ────────────────────────────────────────────

func (c *Client) handleStartGame() {
	room, ok := c.requireHostRoom()
	if !ok {
		return
	}

	if len(room.Players) == 0 {
		c.sendError("NO_PLAYERS", "還沒有玩家加入")
		return
	}

	if err := c.hub.gameService.StartGame(room); err != nil {
		c.sendError("START_FAILED", err.Error())
		return
	}

	c.hub.BroadcastToRoom(room.ID, "GAME_STARTED", map[string]any{
		"roomId":         room.ID,
		"mode":           room.Mode,
		"totalQuestions": len(room.Questions),
	})

	c.hub.startRound(room)
}

func (c *Client) handleEndShooting() {
	room, ok := c.requireHostRoom()
	if !ok {
		return
	}
	c.hub.closeRound(room, "房主提前結束拍照")
}

func (c *Client) handlePickWinners(data any) {
	room, ok := c.requireHostRoom()
	if !ok {
		return
	}

	var req struct {
		PhotoIDs []string `json:"photoIds"`
	}
	if err := decode(data, &req); err != nil {
		c.sendError("BAD_PAYLOAD", "評選參數格式錯誤")
		return
	}

	result, scores, err := c.hub.gameService.ApplyWinners(room, req.PhotoIDs)
	if err != nil {
		c.sendError("PICK_FAILED", err.Error())
		return
	}

	c.hub.BroadcastToRoom(room.ID, "ROUND_RESULT", map[string]any{
		"roomId":        room.ID,
		"questionIndex": result.QuestionIndex,
		"questionText":  result.QuestionText,
		"winners":       result.Winners,
		"groupBonus":    result.GroupBonus,
		"scores":        scores,
		"empty":         false,
	})
}

func (c *Client) handleSkipRound() {
	room, ok := c.requireHostRoom()
	if !ok {
		return
	}

	if room.Status != models.RoomStatusJudging {
		c.sendError("NOT_JUDGING", "現在不是評選階段")
		return
	}

	result := c.hub.gameService.SkipRound(room)
	c.hub.BroadcastToRoom(room.ID, "ROUND_RESULT", map[string]any{
		"roomId":        room.ID,
		"questionIndex": result.QuestionIndex,
		"questionText":  result.QuestionText,
		"winners":       result.Winners,
		"groupBonus":    0,
		"scores":        c.hub.gameService.Leaderboard(room),
		"empty":         true,
	})
}

func (c *Client) handleNextQuestion() {
	room, ok := c.requireHostRoom()
	if !ok {
		return
	}

	if room.Status == models.RoomStatusShooting {
		c.sendError("STILL_SHOOTING", "這題還在拍照中")
		return
	}

	if !c.hub.gameService.NextRound(room) {
		c.hub.finishGame(room)
		return
	}

	c.hub.startRound(room)
}

func (c *Client) handleResetToLobby() {
	room, ok := c.requireHostRoom()
	if !ok {
		return
	}

	c.hub.gameService.ResetToLobby(room)
	c.hub.photoStore.PurgeRoom(room.ID)

	c.hub.BroadcastToRoom(room.ID, "ROOM_RESET_TO_LOBBY", map[string]any{
		"roomId":         room.ID,
		"totalQuestions": len(room.Questions),
		"players":        room.PlayerList(),
	})
}

// ─── 玩家投稿 ────────────────────────────────────────────

func (c *Client) handleSubmitPhoto(data any) {
	if c.RoomID == "" || c.IsHost {
		c.sendError("NOT_PLAYER", "只有玩家可以上傳照片")
		return
	}

	var req struct {
		PhotoID string `json:"photoId"`
	}
	if err := decode(data, &req); err != nil || req.PhotoID == "" {
		c.sendError("BAD_PAYLOAD", "上傳參數格式錯誤")
		return
	}

	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		c.sendError("ROOM_NOT_FOUND", "房間已經不存在了")
		return
	}

	player, ok := room.GetPlayer(c.PlayerID)
	if !ok {
		c.sendError("PLAYER_NOT_FOUND", "你已經不在這個房間裡了")
		return
	}

	// 確認這張照片真的是上傳到這個房間的，避免被塞別房的 photoId
	photo, err := c.hub.photoStore.Get(req.PhotoID)
	if err != nil || photo.RoomID != room.ID {
		c.sendError("PHOTO_NOT_FOUND", "找不到這張照片，請重新拍一次")
		return
	}

	url := "/api/photos/" + photo.ID
	sub, replacedPhotoID, err := c.hub.gameService.AddSubmission(room, player, photo.ID, url)
	if err != nil {
		switch err {
		case services.ErrNotShooting:
			c.sendError("NOT_SHOOTING", "現在不是拍照時間")
		case services.ErrUploadLimit:
			c.sendError("UPLOAD_LIMIT", err.Error())
		default:
			c.sendError("SUBMIT_FAILED", err.Error())
		}
		c.hub.photoStore.Delete(photo.ID)
		return
	}

	replaced := replacedPhotoID != ""
	if replaced {
		c.hub.photoStore.Delete(replacedPhotoID)
	}

	c.sendMessage("PHOTO_ACCEPTED", map[string]any{
		"photoId":  sub.PhotoID,
		"url":      sub.URL,
		"order":    sub.Order,
		"replaced": replaced,
	})

	c.hub.BroadcastToRoom(room.ID, "PHOTO_SUBMITTED", map[string]any{
		"roomId":        room.ID,
		"questionIndex": room.CurrentQuestion,
		"photo":         sub,
		"replaced":      replaced,
		"submitted":     len(room.RoundPhotos),
		"totalPlayers":  room.ConnectedPlayerCount(),
	})

	// 單人模式全員交卷就提前收桌，不用等倒數走完
	if c.hub.gameService.AllPlayersSubmitted(room) {
		c.hub.closeRound(room, "全員都拍完了！")
	}
}
