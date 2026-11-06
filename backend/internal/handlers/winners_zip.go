package handlers

import (
	"archive/zip"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"beauty-game/internal/models"

	"github.com/gin-gonic/gin"
)

var extByType = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
}

// 檔名不能出現的字元（Windows 最嚴格，一起擋掉）
var fileNameReplacer = strings.NewReplacer(
	"/", "／", "\\", "＼", ":", "：", "*", "＊", "?", "？",
	"\"", "＂", "<", "＜", ">", "＞", "|", "｜", "\n", " ", "\r", " ",
)

func safeFileName(s string, maxRunes int) string {
	s = strings.TrimSpace(fileNameReplacer.Replace(s))
	if runes := []rune(s); len(runes) > maxRunes {
		s = string(runes[:maxRunes])
	}
	return s
}

// WinnersZip GET /api/rooms/:roomId/winners.zip
// 把每一題的第 1 名照片打包成一個 zip。檔名帶題號、題目、得獎者，解開來照題號排好就是整場回顧。
// 照片只在記憶體，房間關掉或重置後就下載不到了。
func (h *PhotoHandler) WinnersZip(c *gin.Context) {
	room, err := h.roomService.GetRoom(c.Param("roomId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到這個房間"})
		return
	}

	history := append([]models.RoundResult(nil), room.History...)
	sort.Slice(history, func(i, j int) bool { return history[i].QuestionIndex < history[j].QuestionIndex })

	type entry struct {
		name        string
		contentType string
		data        []byte
	}
	entries := make([]entry, 0, len(history))
	for _, round := range history {
		if len(round.Winners) == 0 {
			continue
		}
		top := round.Winners[0]
		for _, w := range round.Winners {
			if w.Rank < top.Rank {
				top = w
			}
		}
		photo, err := h.store.Get(top.PhotoID)
		if err != nil {
			continue
		}
		ext := extByType[photo.ContentType]
		if ext == "" {
			ext = "jpg"
		}
		name := fmt.Sprintf("第%02d題-%s-%s.%s",
			round.QuestionIndex+1, safeFileName(round.QuestionText, 30), safeFileName(top.PlayerName, 16), ext)
		entries = append(entries, entry{name: name, contentType: photo.ContentType, data: photo.Data})
	}

	if len(entries) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "這場還沒有任何得獎照片"})
		return
	}

	zipName := fmt.Sprintf("今日最美-%s-每題冠軍.zip", room.ID)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="winners-%s.zip"; filename*=UTF-8''%s`,
		room.ID, url.PathEscape(zipName)))
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)

	zw := zip.NewWriter(c.Writer)
	now := time.Now()
	for _, e := range entries {
		// 照片本來就是壓縮過的 JPEG，再 deflate 只是浪費 CPU，直接 Store
		w, err := zw.CreateHeader(&zip.FileHeader{Name: e.name, Method: zip.Store, Modified: now})
		if err != nil {
			return
		}
		if _, err := w.Write(e.data); err != nil {
			return
		}
	}
	_ = zw.Close()
}
