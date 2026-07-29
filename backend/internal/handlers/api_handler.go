package handlers

import (
	"net/http"
	"strings"

	"beauty-game/internal/models"
	"beauty-game/internal/services"
	"beauty-game/internal/websocket"

	"github.com/gin-gonic/gin"
)

// APIHandler 題庫查詢、房間查詢與 WebSocket 升級
type APIHandler struct {
	rooms     *services.RoomService
	questions *services.QuestionService
	hub       *websocket.Hub
}

// NewAPIHandler 建立 API 處理器
func NewAPIHandler(rooms *services.RoomService, questions *services.QuestionService, hub *websocket.Hub) *APIHandler {
	return &APIHandler{rooms: rooms, questions: questions, hub: hub}
}

// GetQuestions GET /api/questions?mode=solo|group
// 給建房頁的題目選擇器列出完整題庫
func (h *APIHandler) GetQuestions(c *gin.Context) {
	mode := models.ModeSolo
	if c.Query("mode") == string(models.ModeGroup) {
		mode = models.ModeGroup
	}

	questions := h.questions.GetBank(mode)
	c.JSON(http.StatusOK, gin.H{
		"mode":      mode,
		"total":     len(questions),
		"questions": questions,
	})
}

// GetRoom GET /api/rooms/:roomId — 加入頁用來確認房號是否有效
func (h *APIHandler) GetRoom(c *gin.Context) {
	room, err := h.rooms.GetRoom(strings.ToUpper(c.Param("roomId")))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到這個房間"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roomId":          room.ID,
		"hostName":        room.HostName,
		"mode":            room.Mode,
		"status":          room.Status,
		"requireNickname": room.RequireNickname,
		"playerCount":     len(room.Players),
		"totalQuestions":  len(room.Questions),
	})
}

// Health GET /api/health
func (h *APIHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "beauty-game-backend",
		"stats":   h.hub.Stats(),
	})
}

// ServeWS GET /ws
func (h *APIHandler) ServeWS(c *gin.Context) {
	websocket.ServeWS(h.hub, c.Writer, c.Request)
}
