package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"beauty-game/internal/models"
	"beauty-game/internal/services"
	"beauty-game/internal/storage"
)

// roomIdleLimit 房間閒置多久沒有任何活動就清掉（連同照片一起釋放）
const roomIdleLimit = 2 * time.Hour

// Hub 集中管理所有連線與房間廣播
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool
	rooms   map[string]map[*Client]bool

	register   chan *Client
	unregister chan *Client

	roomService *services.RoomService
	gameService *services.GameService
	questions   *services.QuestionService
	photoStore  *storage.PhotoStore
	frontendURL string
}

// NewHub 建立 Hub
func NewHub(
	roomService *services.RoomService,
	gameService *services.GameService,
	questions *services.QuestionService,
	photoStore *storage.PhotoStore,
	frontendURL string,
) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		rooms:       make(map[string]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		roomService: roomService,
		gameService: gameService,
		questions:   questions,
		photoStore:  photoStore,
		frontendURL: frontendURL,
	}
}

// questionService 出題服務
func (h *Hub) questionService() *services.QuestionService { return h.questions }

// Run 啟動 Hub 主迴圈與房間清理器
func (h *Hub) Run() {
	go h.roomCleaner()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("🔌 連線建立 client=%s（目前 %d 條）", client.ID, h.ClientCount())

		case client := <-h.unregister:
			h.handleUnregister(client)
		}
	}
}

func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; !ok {
		h.mu.Unlock()
		return
	}
	delete(h.clients, client)

	roomID := client.RoomID
	if roomID != "" {
		if set, ok := h.rooms[roomID]; ok {
			delete(set, client)
			if len(set) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}
	h.mu.Unlock()

	client.closeSend()
	log.Printf("🔌 連線關閉 client=%s room=%s", client.ID, roomID)

	if roomID == "" {
		return
	}

	// 斷線只標記離線、保留分數，等他重新連回來
	if client.IsHost {
		if room, err := h.roomService.GetRoom(roomID); err == nil {
			room.HostConnected = false
		}
		h.BroadcastToRoom(roomID, "HOST_DISCONNECTED", map[string]any{"roomId": roomID})
		return
	}

	if client.PlayerID != "" {
		h.roomService.SetPlayerConnected(roomID, client.PlayerID, false)
		h.broadcastPlayerList(roomID, "PLAYER_DISCONNECTED", map[string]any{
			"playerId": client.PlayerID,
		})
	}
}

// AddClientToRoom 把連線掛到房間廣播群組
func (h *Hub) AddClientToRoom(client *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 換房時先從舊房移除
	if client.RoomID != "" && client.RoomID != roomID {
		if set, ok := h.rooms[client.RoomID]; ok {
			delete(set, client)
			if len(set) == 0 {
				delete(h.rooms, client.RoomID)
			}
		}
	}

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*Client]bool)
	}
	h.rooms[roomID][client] = true
	client.RoomID = roomID
}

// BroadcastToRoom 廣播給房內所有連線
func (h *Hub) BroadcastToRoom(roomID, msgType string, data any) {
	payload, err := json.Marshal(Message{Type: msgType, Data: data})
	if err != nil {
		log.Printf("⚠️ 廣播序列化失敗 type=%s: %v", msgType, err)
		return
	}

	h.mu.RLock()
	targets := make([]*Client, 0, len(h.rooms[roomID]))
	for client := range h.rooms[roomID] {
		targets = append(targets, client)
	}
	h.mu.RUnlock()

	for _, client := range targets {
		client.enqueue(payload)
	}
}

// BroadcastToHost 只送給房主（主畫面）
func (h *Hub) BroadcastToHost(roomID, msgType string, data any) {
	payload, err := json.Marshal(Message{Type: msgType, Data: data})
	if err != nil {
		return
	}

	h.mu.RLock()
	targets := make([]*Client, 0, 1)
	for client := range h.rooms[roomID] {
		if client.IsHost {
			targets = append(targets, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range targets {
		client.enqueue(payload)
	}
}

// broadcastPlayerList 廣播最新玩家名單（附帶額外欄位）
func (h *Hub) broadcastPlayerList(roomID, msgType string, extra map[string]any) {
	room, err := h.roomService.GetRoom(roomID)
	if err != nil {
		return
	}

	data := map[string]any{
		"roomId":  roomID,
		"players": room.PlayerList(),
		"total":   len(room.Players),
	}
	for k, v := range extra {
		data[k] = v
	}

	h.BroadcastToRoom(roomID, msgType, data)
}

// ClientCount 目前連線數
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// BuildJoinURL 產生給 QR code 用的加入連結
func (h *Hub) BuildJoinURL(roomID string) string {
	base := h.frontendURL
	if base == "" {
		return "/join/" + roomID
	}
	return base + "/join/" + roomID
}

// roomCleaner 定期清掉閒置房間，連同它的照片
func (h *Hub) roomCleaner() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		for _, roomID := range h.roomService.ExpiredRooms(roomIdleLimit) {
			h.BroadcastToRoom(roomID, "ROOM_CLOSED", map[string]any{
				"roomId": roomID,
				"reason": "房間閒置太久已自動關閉",
			})

			h.roomService.DeleteRoom(roomID)
			freed := h.photoStore.PurgeRoom(roomID)

			h.mu.Lock()
			delete(h.rooms, roomID)
			h.mu.Unlock()

			log.Printf("🧹 清理閒置房間 %s（釋放 %d 張照片）", roomID, freed)
		}
	}
}

// Stats 服務狀態，給 /api/health 用
func (h *Hub) Stats() map[string]any {
	photos, photoRooms, bytes := h.photoStore.Stats()

	h.mu.RLock()
	rooms := len(h.rooms)
	clients := len(h.clients)
	h.mu.RUnlock()

	return map[string]any{
		"clients":     clients,
		"activeRooms": rooms,
		"totalRooms":  h.roomService.RoomCount(),
		"photos":      photos,
		"photoRooms":  photoRooms,
		"photoBytes":  bytes,
	}
}

// roomSnapshot 組出房間完整狀態，給重連恢復用
func (h *Hub) roomSnapshot(room *models.Room) map[string]any {
	snapshot := map[string]any{
		"roomId":            room.ID,
		"hostName":          room.HostName,
		"mode":              room.Mode,
		"status":            room.Status,
		"requireNickname":   room.RequireNickname,
		"practiceEnabled":   room.PracticeEnabled,
		"inPractice":        room.InPractice,
		"players":           room.PlayerList(),
		"totalQuestions":    len(room.Questions),
		"currentQuestion":   room.CurrentQuestion,
		"questionTimeLimit": room.QuestionTimeLimit,
		"timeLeft":          room.TimeLeft,
		"roundPhotos":       room.RoundPhotos,
		"scores":            room.Leaderboard(),
		"history":           room.History,
		"joinUrl":           h.BuildJoinURL(room.ID),
	}

	if q, ok := room.CurrentQuestionData(); ok {
		snapshot["question"] = q
	}

	return snapshot
}
