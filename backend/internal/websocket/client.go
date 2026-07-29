package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"beauty-game/internal/models"
	"beauty-game/internal/services"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10

	// 照片走 HTTP 上傳，WebSocket 只傳 photoId 與評選清單，8KB 綽綽有餘
	maxMessageSize = 8192
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Message WebSocket 訊息封包
type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Client 一條 WebSocket 連線
type Client struct {
	ID       string
	RoomID   string
	PlayerID string
	IsHost   bool

	conn *websocket.Conn
	hub  *Hub
	send chan []byte

	closeOnce sync.Once
}

// NewClient 建立連線物件
func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:   uuid.New().String(),
		conn: conn,
		hub:  hub,
		send: make(chan []byte, 64),
	}
}

func (c *Client) closeSend() {
	c.closeOnce.Do(func() {
		close(c.send)
	})
}

func (c *Client) enqueue(payload []byte) {
	defer func() {
		// send 已被關閉時忽略，避免對已關閉 channel 寫入而 panic
		_ = recover()
	}()

	select {
	case c.send <- payload:
	default:
		log.Printf("⚠️ client=%s 送出佇列已滿，丟棄訊息", c.ID)
	}
}

// sendMessage 送單一訊息給這條連線
func (c *Client) sendMessage(msgType string, data any) {
	payload, err := json.Marshal(Message{Type: msgType, Data: data})
	if err != nil {
		log.Printf("⚠️ 訊息序列化失敗 type=%s: %v", msgType, err)
		return
	}
	c.enqueue(payload)
}

func (c *Client) sendError(code, message string) {
	c.sendMessage("ERROR", map[string]any{"code": code, "message": message})
}

// decode 把 map 形式的 payload 轉成具體結構
func decode(data any, out any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("⚠️ 讀取失敗 client=%s: %v", c.ID, err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.sendError("BAD_MESSAGE", "訊息格式錯誤")
			continue
		}

		c.handleMessage(&msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case payload, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case "CREATE_ROOM":
		c.handleCreateRoom(msg.Data)
	case "JOIN_ROOM":
		c.handleJoinRoom(msg.Data)
	case "REJOIN_ROOM":
		c.handleRejoinRoom(msg.Data)
	case "CHECK_ROOM":
		c.handleCheckRoom(msg.Data)
	case "RENAME_PLAYER":
		c.handleRename(msg.Data)
	case "UPDATE_SETTINGS":
		c.handleUpdateSettings(msg.Data)
	case "START_GAME":
		c.handleStartGame()
	case "SUBMIT_PHOTO":
		c.handleSubmitPhoto(msg.Data)
	case "END_SHOOTING":
		c.handleEndShooting()
	case "END_PRACTICE":
		c.handleEndPractice()
	case "PICK_WINNERS":
		c.handlePickWinners(msg.Data)
	case "SKIP_ROUND":
		c.handleSkipRound()
	case "NEXT_QUESTION":
		c.handleNextQuestion()
	case "RESET_ROOM_TO_LOBBY":
		c.handleResetToLobby()
	case "LEAVE_ROOM":
		c.handleLeaveRoom()
	case "PING":
		c.sendMessage("PONG", map[string]any{"time": time.Now().UnixMilli()})
	default:
		c.sendError("UNKNOWN_TYPE", "不支援的訊息類型: "+msg.Type)
	}
}

// ─── 房間 ────────────────────────────────────────────────

func (c *Client) handleCreateRoom(data any) {
	var req models.CreateRoomRequest
	if err := decode(data, &req); err != nil {
		c.sendError("BAD_PAYLOAD", "建房參數格式錯誤")
		return
	}

	room, err := c.hub.roomService.CreateRoom(req)
	if err != nil {
		c.sendError("CREATE_FAILED", err.Error())
		return
	}

	room.HostID = c.ID
	c.IsHost = true
	c.PlayerID = c.ID
	c.hub.AddClientToRoom(c, room.ID)

	log.Printf("🏠 開房 %s mode=%s 題數=%d 取名=%v", room.ID, room.Mode, len(room.Questions), room.RequireNickname)

	c.sendMessage("ROOM_CREATED", map[string]any{
		"roomId":            room.ID,
		"hostName":          room.HostName,
		"hostToken":         room.HostToken,
		"clientId":          c.ID,
		"mode":              room.Mode,
		"requireNickname":   room.RequireNickname,
		"practiceEnabled":   room.PracticeEnabled,
		"inPractice":        room.InPractice,
		"totalQuestions":    len(room.Questions),
		"questionTimeLimit": room.QuestionTimeLimit,
		"questions":         room.Questions,
		"joinUrl":           c.hub.BuildJoinURL(room.ID),
	})
}

func (c *Client) handleJoinRoom(data any) {
	var req struct {
		RoomID     string `json:"roomId"`
		PlayerName string `json:"playerName"`
	}
	if err := decode(data, &req); err != nil {
		c.sendError("BAD_PAYLOAD", "加入參數格式錯誤")
		return
	}

	roomID := strings.ToUpper(strings.TrimSpace(req.RoomID))
	room, err := c.hub.roomService.GetRoom(roomID)
	if err != nil {
		c.sendError("ROOM_NOT_FOUND", "找不到這個房間，請確認房號")
		return
	}

	player, err := c.hub.roomService.AddPlayer(roomID, c.ID, req.PlayerName)
	if err != nil {
		switch err {
		case services.ErrNameRequired:
			c.sendError("NAME_REQUIRED", "這個房間需要輸入暱稱")
		case services.ErrNameTaken:
			c.sendError("NAME_TAKEN", "這個暱稱已經有人用了，換一個吧")
		case services.ErrRoomFull:
			c.sendError("ROOM_FULL", "房間人數已滿")
		default:
			c.sendError("JOIN_FAILED", err.Error())
		}
		return
	}

	c.PlayerID = player.ID
	c.IsHost = false
	c.hub.AddClientToRoom(c, room.ID)

	c.sendMessage("JOIN_SUCCESS", map[string]any{
		"roomId":          room.ID,
		"playerId":        player.ID,
		"playerName":      player.Name,
		"avatar":          player.Avatar,
		"mode":            room.Mode,
		"requireNickname": room.RequireNickname,
		"status":          room.Status,
		"totalQuestions":  len(room.Questions),
		"players":         room.PlayerList(),
	})

	c.hub.broadcastPlayerList(room.ID, "PLAYER_JOINED", map[string]any{
		"playerId":   player.ID,
		"playerName": player.Name,
		"avatar":     player.Avatar,
	})

	// 中途加入且遊戲已經在跑：立刻把當前題目補給他
	if room.Status == models.RoomStatusShooting {
		if q, ok := room.CurrentQuestionData(); ok {
			c.sendMessage("NEW_QUESTION", c.hub.questionPayload(room, q))
		}
	}
}

func (c *Client) handleCheckRoom(data any) {
	var req struct {
		RoomID   string `json:"roomId"`
		PlayerID string `json:"playerId"`
	}
	if err := decode(data, &req); err != nil {
		c.sendError("BAD_PAYLOAD", "查詢參數格式錯誤")
		return
	}

	room, err := c.hub.roomService.GetRoom(req.RoomID)
	if err != nil {
		c.sendMessage("ROOM_STATUS", map[string]any{"exists": false, "roomId": req.RoomID})
		return
	}

	_, playerExists := room.GetPlayer(req.PlayerID)
	c.sendMessage("ROOM_STATUS", map[string]any{
		"exists":       true,
		"roomId":       room.ID,
		"status":       room.Status,
		"playerExists": playerExists,
		"isHost":       room.HostID == req.PlayerID,
	})
}

func (c *Client) handleRejoinRoom(data any) {
	var req struct {
		RoomID    string `json:"roomId"`
		PlayerID  string `json:"playerId"`
		HostToken string `json:"hostToken"`
	}
	if err := decode(data, &req); err != nil {
		c.sendError("BAD_PAYLOAD", "重連參數格式錯誤")
		return
	}

	room, err := c.hub.roomService.GetRoom(req.RoomID)
	if err != nil {
		c.sendError("ROOM_NOT_FOUND", "房間已經不存在了")
		return
	}

	// 房主用 hostToken 驗身分，換一條連線也還是房主
	if req.HostToken != "" && req.HostToken == room.HostToken {
		c.IsHost = true
		c.PlayerID = c.ID
		room.HostID = c.ID
		room.HostConnected = true
		c.hub.AddClientToRoom(c, room.ID)

		snapshot := c.hub.roomSnapshot(room)
		snapshot["isHost"] = true
		snapshot["clientId"] = c.ID
		snapshot["hostToken"] = room.HostToken
		c.sendMessage("REJOIN_SUCCESS", snapshot)

		c.hub.BroadcastToRoom(room.ID, "HOST_RECONNECTED", map[string]any{"roomId": room.ID})
		return
	}

	player, ok := room.GetPlayer(req.PlayerID)
	if !ok {
		c.sendError("PLAYER_NOT_FOUND", "你已經不在這個房間裡了")
		return
	}

	c.PlayerID = player.ID
	c.IsHost = false
	c.hub.roomService.SetPlayerConnected(room.ID, player.ID, true)
	c.hub.AddClientToRoom(c, room.ID)

	snapshot := c.hub.roomSnapshot(room)
	snapshot["isHost"] = false
	snapshot["playerId"] = player.ID
	snapshot["playerName"] = player.Name
	snapshot["avatar"] = player.Avatar
	snapshot["hasSubmitted"] = room.HasSubmitted(player.ID)
	c.sendMessage("REJOIN_SUCCESS", snapshot)

	c.hub.broadcastPlayerList(room.ID, "PLAYER_REJOINED", map[string]any{
		"playerId":   player.ID,
		"playerName": player.Name,
	})
}

func (c *Client) handleRename(data any) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decode(data, &req); err != nil || c.RoomID == "" || c.IsHost {
		c.sendError("BAD_PAYLOAD", "改名參數錯誤")
		return
	}

	player, err := c.hub.roomService.RenamePlayer(c.RoomID, c.PlayerID, req.Name)
	if err != nil {
		c.sendError("RENAME_FAILED", err.Error())
		return
	}

	c.sendMessage("RENAME_SUCCESS", map[string]any{"playerId": player.ID, "playerName": player.Name})
	c.hub.broadcastPlayerList(c.RoomID, "PLAYER_RENAMED", map[string]any{
		"playerId":   player.ID,
		"playerName": player.Name,
	})
}

func (c *Client) handleUpdateSettings(data any) {
	room, ok := c.requireHostRoom()
	if !ok {
		return
	}

	var req struct {
		QuestionTimeLimit int      `json:"questionTimeLimit"`
		RequireNickname   *bool    `json:"requireNickname"`
		QuestionMode      string   `json:"questionMode"`
		TotalQuestions    int      `json:"totalQuestions"`
		Difficulty        string   `json:"difficulty"`
		QuestionIDs       []int    `json:"questionIds"`
		CustomQuestions   []string `json:"customQuestions"`
	}
	if err := decode(data, &req); err != nil {
		c.sendError("BAD_PAYLOAD", "設定參數格式錯誤")
		return
	}

	var questions []models.Question
	switch req.QuestionMode {
	case "custom":
		questions = c.hub.questionService().BuildCustom(room.Mode, req.QuestionIDs, req.CustomQuestions)
		if len(questions) == 0 {
			c.sendError("NO_QUESTIONS", "自選題目清單是空的")
			return
		}
	case "random":
		questions = c.hub.questionService().BuildRandom(room.Mode, req.TotalQuestions, req.Difficulty)
	}

	if err := c.hub.roomService.UpdateSettings(room.ID, req.QuestionTimeLimit, req.RequireNickname, questions); err != nil {
		c.sendError("UPDATE_FAILED", err.Error())
		return
	}

	c.hub.BroadcastToRoom(room.ID, "SETTINGS_UPDATED", map[string]any{
		"roomId":            room.ID,
		"questionTimeLimit": room.QuestionTimeLimit,
		"requireNickname":   room.RequireNickname,
		"practiceEnabled":   room.PracticeEnabled,
		"inPractice":        room.InPractice,
		"totalQuestions":    len(room.Questions),
		"questions":         room.Questions,
	})
}

func (c *Client) handleLeaveRoom() {
	if c.RoomID == "" {
		return
	}

	roomID := c.RoomID
	if c.IsHost {
		c.hub.BroadcastToRoom(roomID, "ROOM_CLOSED", map[string]any{
			"roomId": roomID,
			"reason": "房主已關閉房間",
		})
		c.hub.roomService.DeleteRoom(roomID)
		c.hub.photoStore.PurgeRoom(roomID)
		return
	}

	c.hub.roomService.RemovePlayer(roomID, c.PlayerID)
	c.hub.broadcastPlayerList(roomID, "PLAYER_LEFT", map[string]any{"playerId": c.PlayerID})
	c.RoomID = ""
}

// requireHostRoom 取出這條連線的房間，並確認他是房主
func (c *Client) requireHostRoom() (*models.Room, bool) {
	if c.RoomID == "" {
		c.sendError("NOT_IN_ROOM", "你不在任何房間裡")
		return nil, false
	}

	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		c.sendError("ROOM_NOT_FOUND", "房間已經不存在了")
		return nil, false
	}

	if !c.IsHost {
		c.sendError("NOT_HOST", "只有房主可以做這個操作")
		return nil, false
	}

	return room, true
}

// ServeWS 升級 HTTP 連線並啟動讀寫 goroutine
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("⚠️ WebSocket 升級失敗: %v", err)
		return
	}

	client := NewClient(conn, hub)
	hub.register <- client

	go client.writePump()
	go client.readPump()
}
