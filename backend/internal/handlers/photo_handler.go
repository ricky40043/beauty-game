package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"beauty-game/internal/services"
	"beauty-game/internal/storage"

	"github.com/gin-gonic/gin"
)

// PhotoHandler 照片上傳與讀取
type PhotoHandler struct {
	store       *storage.PhotoStore
	roomService *services.RoomService
	maxBytes    int64
}

// NewPhotoHandler 建立照片處理器
func NewPhotoHandler(store *storage.PhotoStore, roomService *services.RoomService, maxBytes int64) *PhotoHandler {
	return &PhotoHandler{store: store, roomService: roomService, maxBytes: maxBytes}
}

// Upload POST /api/rooms/:roomId/photos
// multipart form，欄位名 photo。回傳 { photoId, url }，接著由前端用 WebSocket 送 SUBMIT_PHOTO。
func (h *PhotoHandler) Upload(c *gin.Context) {
	roomID := strings.ToUpper(strings.TrimSpace(c.Param("roomId")))

	if !h.roomService.Exists(roomID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到這個房間"})
		return
	}

	// 先擋掉超大 body，避免整包讀進記憶體
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBytes+1024)

	file, err := c.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "沒有收到照片檔案"})
		return
	}
	if file.Size > h.maxBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "照片太大了，請重拍"})
		return
	}

	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "讀取照片失敗"})
		return
	}
	defer opened.Close()

	data, err := io.ReadAll(io.LimitReader(opened, h.maxBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "讀取照片失敗"})
		return
	}

	photo, err := h.store.Put(roomID, data)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, storage.ErrRoomQuota) {
			status = http.StatusInsufficientStorage
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"photoId": photo.ID,
		"url":     "/api/photos/" + photo.ID,
		"size":    len(photo.Data),
	})
}

// Get GET /api/photos/:photoId
func (h *PhotoHandler) Get(c *gin.Context) {
	photo, err := h.store.Get(c.Param("photoId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到這張照片"})
		return
	}

	// 照片只存在記憶體、id 是隨機 uuid，內容不會變，讓瀏覽器盡量快取
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "inline")
	c.Data(http.StatusOK, photo.ContentType, photo.Data)
}
